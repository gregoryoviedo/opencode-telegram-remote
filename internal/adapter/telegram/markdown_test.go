package telegram

import "testing"

func TestMarkdownToTelegramHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "plain text passes through",
			in:   "Hello world",
			want: "Hello world",
		},
		{
			name: "bold becomes tag",
			in:   "**Astro** 7.1 + **Tailwind CSS** 4",
			want: "<b>Astro</b> 7.1 + <b>Tailwind CSS</b> 4",
		},
		{
			name: "inline code becomes tag",
			in:   "Use `pnpm` to install",
			want: "Use <code>pnpm</code> to install",
		},
		{
			name: "code block with language hint",
			in:   "```go\nfunc main() {}\n```",
			want: "<pre>func main() {}</pre>",
		},
		{
			name: "code block without language hint",
			in:   "```\nplain text\n```",
			want: "<pre>plain text</pre>",
		},
		{
			name: "html special chars are escaped",
			in:   "5 < 10 & ok",
			want: "5 &lt; 10 &amp; ok",
		},
		{
			name: "http link becomes anchor",
			in:   "Open [Google](https://google.com) now",
			want: `Open <a href="https://google.com">Google</a> now`,
		},
		{
			name: "link without protocol is left alone",
			in:   "See [docs](internal/spec)",
			want: "See [docs](internal/spec)",
		},
		{
			name: "header becomes bold",
			in:   "# Title\n\nBody",
			want: "<b>Title</b>\n\nBody",
		},
		{
			name: "italic word between spaces",
			in:   "a *real* one",
			want: "a <i>real</i> one",
		},
		{
			name: "bullet dash becomes dot",
			in:   "- first\n- second",
			want: "• first\n• second",
		},
		{
			name: "bullet star becomes dot and is not italic",
			in:   "* first\n* second",
			want: "• first\n• second",
		},
		{
			name: "opencode sample output",
			in: "**Astro** 7.1 + **Tailwind CSS** 4 (via @tailwindcss/vite). con:\n" +
				"- `astro-icon` + `@iconify-json/lucide` → iconos Lucide\n" +
				"- `@astrojs/sitemap` con i18n\n" +
				"- `i18next` para internacionalización (en, es, de, fr, pt, ja)\n" +
				"- TypeScript, ESM, package manager `pnpm`\n" +
				"- Sitio: `gregoryoviedo.com`",
			want: "<b>Astro</b> 7.1 + <b>Tailwind CSS</b> 4 (via @tailwindcss/vite). con:\n" +
				"• <code>astro-icon</code> + <code>@iconify-json/lucide</code> → iconos Lucide\n" +
				"• <code>@astrojs/sitemap</code> con i18n\n" +
				"• <code>i18next</code> para internacionalización (en, es, de, fr, pt, ja)\n" +
				"• TypeScript, ESM, package manager <code>pnpm</code>\n" +
				"• Sitio: <code>gregoryoviedo.com</code>",
		},
		{
			name: "commit style message with backticks",
			in:   "`42efca9` — 2026-08-10 16:13 -0400 — feat: add Python skill asset",
			want: "<code>42efca9</code> — 2026-08-10 16:13 -0400 — feat: add Python skill asset",
		},
		{
			name: "bold inside inline code is preserved",
			in:   "Run `npm **install**`",
			want: "Run <code>npm **install**</code>",
		},
		{
			name: "unclosed asterisks left alone",
			in:   "this *is not italic",
			want: "this *is not italic",
		},
		{
			name: "double dash is not a bullet",
			in:   "use --debug flag",
			want: "use --debug flag",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := markdownToTelegramHTML(tc.in)
			if got != tc.want {
				t.Errorf("markdownToTelegramHTML(%q)\n got: %q\nwant: %q", tc.in, got, tc.want)
			}
		})
	}
}
