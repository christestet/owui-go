// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

// Project page on GitHub Pages is served from a sub-path, so `base` must match
// the repository name. If you later add a custom domain, set `site` to it and
// change `base` to "/".
export default defineConfig({
  site: "https://christestet.github.io",
  base: "/owui-go",
  integrations: [
    starlight({
      title: "owui",
      description:
        "A fast and flexible CLI written in Go for managing multiple Open WebUI instances.",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/christestet/owui-go",
        },
      ],
      editLink: {
        baseUrl: "https://github.com/christestet/owui-go/edit/main/docs/",
      },
      lastUpdated: true,
      tableOfContents: { minHeadingLevel: 2, maxHeadingLevel: 3 },
      sidebar: [
        {
          label: "Guide",
          items: [
            { label: "Overview", slug: "overview" },
            { label: "Install", slug: "install" },
            { label: "Usage", slug: "usage" },
            { label: "Configuration", slug: "configuration" },
          ],
        },
        {
          label: "Reference",
          items: [{ autogenerate: { directory: "reference/cli" } }],
        },
      ],
    }),
  ],
});
