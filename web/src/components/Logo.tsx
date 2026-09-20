export default function Logo({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" aria-hidden>
      <path d="M16 2.5 29 16 16 29.5 3 16 16 2.5Z" fill="#E24C2C" />
      <path d="M16 3.2 16 28.8M4.2 16H27.8" stroke="#FFF8F0" strokeWidth="1.6" />
    </svg>
  );
}
