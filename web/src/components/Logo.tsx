type LogoProps = {
  variant?: "color" | "white";
  className?: string;
  priority?: boolean;
  /** Rendered height in CSS pixels. Official mark is 1024×679. */
  height?: number;
};

/** Official mark from tiper.co.tz (Tiper-Main-Logo). */
const LOGO_SRC = "/tiper-logo.png";
const ASPECT = 1024 / 679;

export function Logo({
  variant = "color",
  className = "",
  height = 48,
  priority,
}: LogoProps) {
  const width = Math.round(height * ASPECT);
  const frame =
    variant === "white" ? "rounded-md bg-white px-1.5 py-1 shadow-sm" : "";
  return (
    <span className={`inline-flex items-center ${frame} ${className}`}>
      {/* Native img: next/image + h-auto/w-auto collapsed the mark to 0×0. */}
      <img
        src={LOGO_SRC}
        alt="TIPER — Tanzania International Petroleum Reserves Limited"
        width={width}
        height={height}
        decoding="async"
        fetchPriority={priority ? "high" : "auto"}
        className="block object-contain"
        style={{ height, width }}
      />
    </span>
  );
}
