<script>
  const raw = String(__APP_VERSION__ || 'dev').replace(/^v/, '');
  const versionLabel = raw === 'dev' ? 'dev' : `v${raw}`;
</script>

<footer class="footer sm:footer-horizontal bg-base-100 items-center p-4">
  <aside class="grid-flow-col items-center">
    <img
      src="/favicon.svg"
      alt="SOP and GO logo"
      class="h-12 w-12"
      title="Based on the Go Gopher by Renee French (CC BY 4.0)" 
    />
    <p>
      <span class="font-semibold">
        SOP and GO
      </span>
      <br />
      © {new Date().getFullYear()} Leopold Dürrauer · {versionLabel} · MIT License
    </p>
  </aside>
  <nav class="grid-flow-col gap-4 md:place-self-center md:justify-self-end">
    <a
      href="https://duerrauer.com"
      target="_blank"
      rel="noopener noreferrer"
      aria-label="Dr. Leopold Dürrauer Website"
    >
       <img
        src="/duerrauer.svg"
        alt="Dürrauer logo"
        class="h-8 w-8"
      />
    </a>
    <a
      href="https://github.com/Sopandgo/sopandgo"
      target="_blank"
      rel="noopener noreferrer"
      aria-label="GitHub repository"
    >
       <img
        src="/github.svg"
        alt="GitHub logo"
        class="h-8 w-8"
      />
    </a>    
  </nav>
</footer>