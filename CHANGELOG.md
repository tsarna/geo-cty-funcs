# Changelog

## v0.2.0

### Added

- `geo_point` now accepts a single combined coordinate string, parsing all
  five `geo_format` output formats: `"lat,lon"`, `"lat,lon,alt"`,
  DMS with hemisphere, DMS ASCII, and DMS signed. This enables round-tripping
  through `geo_point(geo_format(p, fmt))`.
- `geo_point(combined, base)` form for merging a combined string onto a base object.

### Fixed

- Clarified polar region behavior in documentation for solar functions.

## v0.1.1

### Fixed

- Fix solar midnight (and other cross-day events) returning a result ~25 hours
  away instead of the imminent occurrence when called near the UTC date boundary.
  The search now starts one day earlier to catch events spanning two calendar days.

## v0.1.0

- Initial release with 16 functions:
  - Point construction and formatting: `geo_point`, `geo_format`
  - Solar time: `sunrise`, `sunset`, `solar_noon`, `solar_midnight`
  - Celestial position: `sun_position`, `moon_position`, `moon_phase`
  - Geodesic (WGS-84): `geo_destination`, `geo_inverse`, `geo_waypoints`
  - Geometric (S2): `geo_area`, `geo_contains`, `geo_nearest`, `geo_line_intersect`
