<script lang="ts">
  import * as m from '#lib/paraglide/messages.js';

  const raw = String(__APP_VERSION__ || 'dev').replace(/^v/, '');
  const versionLabel = raw === 'dev' ? 'dev' : `v${raw}`;
</script>

<footer class="footer sm:footer-horizontal items-center border-t border-base-300 bg-base-100 p-4">
  <aside class="grid-flow-col items-center">
    <img
      src="/favicon.svg"
      alt={m.footer_logo_alt()}
      class="h-12 w-12"
      title={m.footer_gopher()}
    />
    <p>
      <span class="font-semibold">
        SOP and GO
      </span>
      <br />
      © {new Date().getFullYear()} Leopold Dürrauer · {versionLabel} · {m.footer_license()}
    </p>
  </aside>
  <nav class="grid-flow-col gap-4 md:place-self-center md:justify-self-end">
    <a
      href="https://duerrauer.com"
      target="_blank"
      rel="noopener noreferrer"
      aria-label={m.footer_author_site()}
    >
       <img
        src="/duerrauer.svg"
        alt={m.footer_duerrauer_alt()}
        class="h-8 w-8"
      />
    </a>
    <a
      href="https://github.com/Sopandgo/sopandgo"
      target="_blank"
      rel="noopener noreferrer"
      aria-label={m.footer_github()}
    >
       <img
        src="/github.svg"
        alt={m.footer_github_alt()}
        class="h-8 w-8"
      />
    </a>    
  </nav>
</footer>
