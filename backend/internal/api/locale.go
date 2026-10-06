package api

import "github.com/sopandgo/sopandgo/backend/internal/i18n"

func (s *Server) orgLocale() string {
	if s.smtpSettings == nil {
		return i18n.Base
	}
	tag, err := s.smtpSettings.GetDefaultLocale()
	if err != nil {
		return i18n.Base
	}
	return tag
}
