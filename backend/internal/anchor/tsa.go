package anchor

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/digitorus/pkcs7"
	"github.com/digitorus/timestamp"
)

// Auth types for the TSA.
const (
	AuthNone   = "none"
	AuthBasic  = "basic"
	AuthBearer = "bearer"
)

// maxTSAResponse bounds what we read from a TSA. Real responses are a few KB.
const maxTSAResponse = 1 << 20

// TSA talks to one RFC 3161 Time Stamping Authority.
type TSA struct {
	URL      string
	AuthType string
	Username string
	Secret   string // password for basic, token for bearer
	Client   *http.Client
}

// ValidateTSAURL accepts absolute http(s) URLs. Plain http is allowed because
// many public TSAs only offer it; the response is signed, so transport security
// is not what protects it.
func ValidateTSAURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("TSA URL must be an absolute http or https URL")
	}
	if u.User != nil {
		return fmt.Errorf("put TSA credentials in the credential fields, not the URL")
	}
	return nil
}

// Stamp asks the TSA to timestamp digest (a SHA-256 hash) and returns the raw
// DER TimeStampResp. It checks only that the TSA granted the request and that
// the token answers this request (digest and nonce); VerifyReceipt checks who
// signed it.
func (t *TSA) Stamp(ctx context.Context, digest []byte) ([]byte, *timestamp.Timestamp, error) {
	if len(digest) != sha256.Size {
		return nil, nil, fmt.Errorf("digest must be %d bytes", sha256.Size)
	}
	nonce, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		return nil, nil, err
	}
	req := timestamp.Request{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: digest,
		Certificates:  true, // ask the TSA to include its signing certificate
		Nonce:         nonce,
	}
	body, err := req.Marshal()
	if err != nil {
		return nil, nil, fmt.Errorf("build timestamp request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(t.URL), bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/timestamp-query")
	httpReq.Header.Set("Accept", "application/timestamp-reply")
	switch t.AuthType {
	case AuthBasic:
		httpReq.SetBasicAuth(t.Username, t.Secret)
	case AuthBearer:
		httpReq.Header.Set("Authorization", "Bearer "+t.Secret)
	}

	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("TSA request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxTSAResponse+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read TSA response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("TSA returned HTTP %d", resp.StatusCode)
	}
	if len(raw) > maxTSAResponse {
		return nil, nil, fmt.Errorf("TSA response is larger than %d bytes", maxTSAResponse)
	}

	ts, err := timestamp.ParseResponse(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("TSA response: %w", err)
	}
	if ts.Nonce == nil || ts.Nonce.Cmp(nonce) != 0 {
		return nil, nil, errors.New("TSA response does not answer this request (nonce mismatch)")
	}
	if ts.HashAlgorithm != crypto.SHA256 || !bytes.Equal(ts.HashedMessage, digest) {
		return nil, nil, errors.New("TSA timestamped a different hash than requested")
	}
	return raw, ts, nil
}

// VerifyReceipt parses a stored TimeStampResp and checks that it timestamps
// digest with SHA-256 and that its signing certificate chains to one of the
// pinned certificates, with the time-stamping key usage, at the token's time.
func VerifyReceipt(raw, digest []byte, pinned []*x509.Certificate) (*timestamp.Timestamp, error) {
	ts, err := timestamp.ParseResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("receipt does not parse or its signature is invalid: %w", err)
	}
	if ts.HashAlgorithm != crypto.SHA256 || !bytes.Equal(ts.HashedMessage, digest) {
		return nil, errors.New("receipt timestamps a different hash")
	}
	if err := verifySigner(ts, pinned); err != nil {
		return nil, err
	}
	return ts, nil
}

// signerCert returns the certificate that signed the token and the other
// certificates the TSA sent. ParseResponse has already checked the CMS
// signature with that signer.
func signerCert(ts *timestamp.Timestamp) (*x509.Certificate, []*x509.Certificate, error) {
	if len(ts.Certificates) == 0 {
		return nil, nil, errors.New("receipt does not include the TSA certificate")
	}
	p7, err := pkcs7.Parse(ts.RawToken)
	if err != nil {
		return nil, nil, err
	}
	leaf := p7.GetOnlySigner()
	if leaf == nil {
		return nil, nil, errors.New("receipt must have exactly one signer")
	}
	var others []*x509.Certificate
	for _, c := range ts.Certificates {
		if !bytes.Equal(c.Raw, leaf.Raw) {
			others = append(others, c)
		}
	}
	return leaf, others, nil
}

func verifySigner(ts *timestamp.Timestamp, pinned []*x509.Certificate) error {
	if len(pinned) == 0 {
		return errors.New("no TSA certificate is pinned")
	}
	leaf, others, err := signerCert(ts)
	if err != nil {
		return err
	}
	if err := checkTSAKeyUsage(leaf); err != nil {
		return err
	}
	roots := x509.NewCertPool()
	for _, c := range pinned {
		roots.AddCert(c)
	}
	inter := x509.NewCertPool()
	for _, c := range others {
		inter.AddCert(c)
	}
	_, err = leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: inter,
		CurrentTime:   ts.Time,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	})
	if err != nil {
		return fmt.Errorf("TSA certificate does not chain to the pinned CA: %w", err)
	}
	return nil
}

var oidExtKeyUsage = asn1.ObjectIdentifier{2, 5, 29, 37}

// checkTSAKeyUsage applies RFC 3161 section 2.3: the TSA certificate's extended
// key usage must be critical and list only time-stamping. OpenSSL's
// "openssl ts -verify" enforces this too, so a receipt SOPandGO accepts is one
// an auditor's OpenSSL accepts.
func checkTSAKeyUsage(leaf *x509.Certificate) error {
	critical := false
	for _, ext := range leaf.Extensions {
		if ext.Id.Equal(oidExtKeyUsage) {
			critical = ext.Critical
		}
	}
	if !critical || len(leaf.ExtKeyUsage) != 1 || leaf.ExtKeyUsage[0] != x509.ExtKeyUsageTimeStamping || len(leaf.UnknownExtKeyUsage) != 0 {
		return errors.New("TSA certificate must have a critical extended key usage of time-stamping only (RFC 3161)")
	}
	return nil
}

// PinFromToken picks the certificates to pin from a probe token: every
// certificate the TSA sent except its own signing certificate, so routine
// leaf rotation keeps working. When the TSA sends only its leaf, the issuer is
// fetched from the leaf's Authority Information Access URL.
func PinFromToken(ctx context.Context, ts *timestamp.Timestamp, client *http.Client) ([]*x509.Certificate, error) {
	leaf, others, err := signerCert(ts)
	if err != nil {
		return nil, err
	}
	if len(others) > 0 {
		return others, nil
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	for _, u := range leaf.IssuingCertificateURL {
		issuer, err := fetchCert(ctx, client, u)
		if err == nil && leaf.CheckSignatureFrom(issuer) == nil {
			return []*x509.Certificate{issuer}, nil
		}
	}
	return nil, errors.New("the TSA sent only its own certificate and its issuer could not be fetched; paste the TSA's CA certificate instead")
}

func fetchCert(ctx context.Context, client *http.Client, u string) (*x509.Certificate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxTSAResponse))
	if err != nil {
		return nil, err
	}
	if certs, err := ParsePEMCerts(string(b)); err == nil && len(certs) > 0 {
		return certs[0], nil
	}
	return x509.ParseCertificate(b)
}

// ParsePEMCerts parses one or more PEM certificates.
func ParsePEMCerts(s string) ([]*x509.Certificate, error) {
	var out []*x509.Certificate
	rest := []byte(s)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, errors.New("no PEM certificates found")
	}
	return out, nil
}

// EncodePEMCerts writes certificates as concatenated PEM.
func EncodePEMCerts(certs []*x509.Certificate) string {
	var b strings.Builder
	for _, c := range certs {
		_ = pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
	}
	return b.String()
}

// Fingerprint is the SHA-256 of the first pinned certificate, as colon-free
// lowercase hex, for admins to compare with what their TSA publishes.
func Fingerprint(certs []*x509.Certificate) string {
	if len(certs) == 0 {
		return ""
	}
	sum := sha256.Sum256(certs[0].Raw)
	return hex.EncodeToString(sum[:])
}
