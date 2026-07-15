# Changelog

## v0.4.0

### Changed

- **BREAKING: the functions are namespaced.** The geographic, geodesic, and geometric
  functions — whose bare names (`point`, `area`, `contains`, `nearest`) would be too
  generic to expose globally — move under `geo::`; the solar-event and celestial-position
  functions move under `sky::`. HCL parses `a::b(x)` natively as a single flat map key, so
  this is a naming change, not a structural one, and the leaf names shed the prefix the
  flat names carried. **Existing `.vcl`/`.cty` files must be updated.**

  | was | is |
  | --- | --- |
  | `geo_point`, `geo_format` | `geo::point`, `geo::format` |
  | `geo_inverse`, `geo_destination`, `geo_waypoints` | `geo::inverse`, `geo::destination`, `geo::waypoints` |
  | `geo_area`, `geo_contains`, `geo_nearest`, `geo_line_intersect` | `geo::area`, `geo::contains`, `geo::nearest`, `geo::line_intersect` |
  | `sunrise`, `sunset`, `solar_noon`, `solar_midnight` | `sky::sunrise`, `sky::sunset`, `sky::solar_noon`, `sky::solar_midnight` |
  | `sun_position`, `moon_position`, `moon_phase` | `sky::sun_position`, `sky::moon_position`, `sky::moon_phase` |

- `Externs()` now returns `map[string][]byte` (one file per namespace: `externs/geo.cty`,
  `externs/sky.cty`) rather than a single `[]byte`, and `ExternsFilename` is removed — a
  functy source declares at most one namespace. A host registers them with a loop:

  ```go
  for name, src := range geocty.Externs() {
      parser.RegisterExterns(src, name)
  }
  ```

  The `geopoint` type is unchanged and stays un-namespaced (it is a type, not a function).

## v0.3.0

### Added

- **`IsGeoPoint` and the `geopoint` type.** `IsGeoPoint(cty.Value) error` is the
  predicate for "an object with numeric `lat` and `lon`" — extra attributes (altitude,
  time, speed, name) allowed and passed through. A functy host registers it as the open
  type `geopoint`, so a `.cty` annotation, a signature, or a host `var` constraint can
  name the shape every geo function speaks:

  ```go
  parser.RegisterOpenType("geopoint", geocty.IsGeoPoint)
  ```

- **`Externs()` — the real signatures, for functy hosts.** `externs.cty` (embedded;
  exposed as opaque bytes via `Externs()` and `ExternsFilename`) declares every geo
  function as a [functy](https://github.com/tsarna/functy) `//functy:extern` declaration.
  It is never compiled and declares nothing callable; it exists so that `help()`,
  generated documentation, and editor tooling can show what cty metadata cannot — for two
  distinct reasons:

  - Some functions fake an optional or type-dispatched argument with a variadic
    (`geo_format`'s format, the solar functions' offset/time, `sun_position`'s time), or
    vary their result shape (`geo_point` merging a base). cty cannot describe these at all.
  - The rest have a fixed arity and return shape, but their point arguments are objects
    with arbitrary extra attributes, so their cty parameter type is `dynamic` — and a
    dynamic argument poisons cty's return type to `dynamic`, hiding the whole return
    payload. `geo_inverse`'s `{distance, bearing, back_bearing}` was invisible from
    metadata alone. The declaration restores it, and types the inputs as `geopoint`.

  This package does not import functy; the bytes are opaque to it.

- Every function and every parameter now carries a cty `Description`. The metadata is the
  only documentation a non-functy cty host can see, and the only thing functy's own
  `doc()` reads.

### Changed

- Depends on `time-cty-funcs` v0.4.0 (was v0.2.0), whose time/duration functions are now
  namespaced (`time::*`, `duration::*`). The capsule types and helper functions this
  package uses (`TimeCapsuleType`, `GetTime`, `NewTimeCapsule`, …) are unchanged.

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
