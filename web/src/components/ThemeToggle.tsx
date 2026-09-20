import { useTheme } from "../theme";

export default function ThemeToggle() {
  const { theme, toggle } = useTheme();
  const dark = theme === "dark";
  return (
    <button
      type="button"
      className="btn ghost theme-toggle"
      onClick={toggle}
      aria-label={dark ? "Включить светлую тему" : "Включить тёмную тему"}
      title={dark ? "Светлая" : "Тёмная"}
    >
      {dark ? "Светлая" : "Тёмная"}
    </button>
  );
}
