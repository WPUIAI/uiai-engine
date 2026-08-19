// Tailwind tokens aligned with shadcn-svelte + Bits UI conventions.
// Central tokens live in src/lib/ui/design-tokens.css as CSS variables.
export default {
  content: ["./src/**/*.{svelte,ts,js}", "./src/**/*.svelte"],
  theme: {
    extend: {
      colors: {
        bg: "var(--color-bg)",
        surface: "var(--color-surface)",
        "surface-elevated": "var(--color-surface-elevated)",
        text: "var(--color-text)",
        "text-muted": "var(--color-text-muted)",
        border: "var(--color-border)",
        "border-strong": "var(--color-border-strong)",
        accent: "var(--color-accent)",
        success: "var(--color-success)",
        warn: "var(--color-warn)",
        error: "var(--color-error)",
      },
      borderRadius: {
        card: "var(--radius-card)",
        button: "var(--radius-button)",
        input: "var(--radius-input)",
        pill: "var(--radius-pill)",
      },
      boxShadow: {
        card: "var(--shadow-card)",
        overlay: "var(--shadow-overlay)",
        popover: "var(--shadow-popover)",
      },
      fontFamily: {
        display: "var(--font-display)",
        body: "var(--font-body)",
      },
    },
  },
  plugins: [],
};
