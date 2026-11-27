// A management UI screenshot with a caption. Images are captured at 2x, so width/height are
// the display size (half the file's pixel size) to reserve layout space before they load.
// Use `narrow` for side-panel shots so they are not blown up to the full column width.
export default function Screenshot({ src, alt, width, height, caption, narrow = false }) {
  return (
    <figure className={narrow ? "docs-screenshot docs-screenshot-narrow" : "docs-screenshot"}>
      <img src={src} alt={alt} width={width} height={height} loading="lazy" decoding="async" />
      {caption ? <figcaption>{caption}</figcaption> : null}
    </figure>
  );
}
