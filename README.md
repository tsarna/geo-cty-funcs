# geo-cty-funcs

A Go module providing geographic, solar/astronomical, geodesic, and geometric
functions for use in [go-cty](https://github.com/zclconf/go-cty) / HCL2
evaluation contexts.

## Installation

```
go get github.com/tsarna/geo-cty-funcs
```

## Usage

```go
import (
    geocty "github.com/tsarna/geo-cty-funcs"
    "github.com/zclconf/go-cty/cty/function"
)

// Register all functions in an HCL eval context
funcs := geocty.GetGeoFunctions()
// funcs is map[string]function.Function — merge into your eval context
```

## The `point` Object

A `point` is a plain cty object with at least `lat` and `lon` number
attributes (signed decimal degrees). Additional fields (e.g. `alt`, `speed`,
`track`, `time`) are preserved by functions that return derived points. The
field names are aligned with the
[GPSD JSON `TPV` record](https://gpsd.gitlab.io/gpsd/gpsd_json.html), so data
from a GPSD source can flow through without translation.

```hcl
p = {lat = 37.7749, lon = -122.4194}
p2 = {lat = 51.5074, lon = -0.1278, alt = 11.0, speed = 10.0}
```

`IsGeoPoint(cty.Value) error` is that shape as a predicate — nil when the value is an
object with numeric `lat` and `lon`, an error otherwise. A [functy](https://github.com/tsarna/functy)
host registers it as the open type `geopoint`, so a `.cty` annotation or a host variable
constraint can require a point while letting the extra fields ride along:

```go
parser.RegisterOpenType("geopoint", geocty.IsGeoPoint)
```

## Signature declarations

The signatures these functions really have are not the ones cty can express, for two
reasons. Some fake an optional or type-dispatched argument with a variadic — `geo::format`'s
format, the solar functions' offset and time, the union third argument of `geo::destination`
— or vary their result shape, as `geo::point` does when it merges a base; cty cannot describe
these at all. The rest have a fixed arity and a fixed return shape, but their point arguments
are objects with arbitrary extras, so their cty parameter type is `dynamic` — and a dynamic
argument poisons cty's return type to `dynamic` too, hiding the whole return payload
(`geo::inverse`'s `{distance, bearing, back_bearing}` reflects as nothing).

So `externs/geo.cty` and `externs/sky.cty` declare every function as a functy
`//functy:extern` declaration — one file per namespace — typing the point arguments as
`geopoint` and spelling out each return shape. They are never compiled and declare nothing
callable; they exist so `help()`, generated documentation, and editor tooling can show the
real signatures. `Externs()` returns them as opaque bytes keyed by filename — this package
does not import functy:

```go
parser.RegisterOpenType("geopoint", geocty.IsGeoPoint)
for name, src := range geocty.Externs() {
    parser.RegisterExterns(src, name)
}
```

A host that is not a functy host can ignore both; the cty `Description` on every function
and parameter is still populated.

## Functions

### Point Construction and Formatting

| Function | Signature | Description |
|----------|-----------|-------------|
| `geo::point` | `geo::point(combined)` or `geo::point(lat, lon[, base])` | Constructs a point from a combined string or separate lat/lon |
| `geo::format` | `geo::format(point[, format])` | Formats a point as a string |

#### `geo::point(combined[, base])` or `geo::point(lat, lon[, base])`

A single combined string is parsed by splitting on `,` (decimal formats) or on
the hemisphere-letter boundary (DMS formats). All `geo::format` output formats
round-trip through the single-string form.

When two separate values are given, `lat` and `lon` may each be a **number**
(signed decimal degrees) or a **string** in any of these forms:

- Signed decimal: `"37.7749"`, `"-122.4194"`
- Decimal with hemisphere: `"37.7749 N"`, `"N 37.7749"`, `"122.4194W"`
- DMS with degree symbol: `"37°46'29.6"N"`
- DMS with ASCII `d`/`D`: `"37d46'29.6"N"`
- Whitespace-separated DMS: `"37 46 29.6 N"`

If `base` is supplied, the result is a copy of `base` with `lat`/`lon` overwritten.

```hcl
geo::point("37.7749,-122.4194")
geo::point("37°46'29\"N 122°25'9\"W")
geo::point(37.7749, -122.4194)
geo::point("37°46'29\"N", "122°25'9\"W")
geo::point(37.7749, -122.4194, {alt = 11.0, track = 90.0})
geo::point("37.7749,-122.4194", {alt = 11.0})
```

#### `geo::format(point[, format])`

| Format | Example |
|--------|---------|
| `"decimal"` (default) | `"37.7749,-122.4194"` |
| `"decimal_alt"` | `"37.7749,-122.4194,11"` (includes `alt` if present) |
| `"dms"` | `"37°46'29.6"N 122°25'9.8"W"` |
| `"dms_ascii"` | `"37d46'29.6"N 122d25'9.8"W"` |
| `"dms_signed"` | `"37°46'29.6" -122°25'9.8""` |

---

### Solar / Astronomical Time Functions

All four solar functions share the same signature pattern:

```
f(point)
f(point, offset)     — offset is a duration capsule
f(point, t)          — t is a time capsule
f(point, offset, t)
```

They return the **next occurrence** after `t` (default: `now()`) of the solar
event plus `offset` (default: zero). If the computed target is in the past,
the function advances to the next day. All times are UTC.

In **polar regions** where the sun does not rise or set for extended periods,
the functions search forward day-by-day (up to 400 days) until the event
occurs again. `sky::solar_noon` and `sky::solar_midnight` require both sunrise and
sunset to exist on the relevant day(s); during polar day or polar night they
return the first meaningful occurrence after polar conditions end.

| Function | Description |
|----------|-------------|
| `sky::sunrise` | Next sunrise |
| `sky::sunset` | Next sunset |
| `sky::solar_noon` | Next solar noon (midpoint of sunrise and sunset) |
| `sky::solar_midnight` | Next solar midnight (midpoint of sunset and next sunrise) |

```hcl
time = sky::sunrise(var.location)
time = sky::sunrise(var.location, duration("-30m"))
time = sky::sunset(var.location, duration("-15m"), ctx.scheduled_time)
```

---

### Sun and Moon Position

| Function | Signature | Returns |
|----------|-----------|---------|
| `sky::sun_position` | `sky::sun_position(point[, t])` | `{azimuth, altitude}` (degrees) |
| `sky::moon_position` | `sky::moon_position(point[, t])` | `{azimuth, altitude, distance}` (degrees, meters) |
| `sky::moon_phase` | `sky::moon_phase([t])` | `{fraction, phase, angle}` |

- **azimuth**: clockwise from true north (0=N, 90=E, 180=S, 270=W)
- **altitude**: degrees above horizon (negative = below)
- **distance**: meters to the moon's centre
- **fraction**: 0-1, illuminated fraction of the visible disc
- **phase**: 0-1, position in lunation cycle (0 = new, 0.5 = full)
- **angle**: bright-limb angle in degrees (sign distinguishes waxing/waning)

```hcl
s = sky::sun_position(var.location)
s.altitude > 0            # true if the sun is up
s.altitude < -6.0         # true if past civil twilight

mp = sky::moon_phase()
mp.phase < 0.5            # true if waxing
```

---

### Geodesic Functions (WGS-84)

| Function | Signature | Description |
|----------|-----------|-------------|
| `geo::inverse` | `geo::inverse(a, b)` | Distance and bearings between two points |
| `geo::destination` | `geo::destination(origin, bearing, third[, extras])` | Point reached by travelling on a bearing |
| `geo::waypoints` | `geo::waypoints(a, b, n)` | n evenly spaced points along the geodesic |

#### `geo::inverse(point_a, point_b)`

Returns `{distance, bearing, back_bearing}`. Distance in meters, bearings in
degrees clockwise from north.

```hcl
r = geo::inverse(var.current, var.destination)
r.distance    # meters
r.bearing     # heading toward destination
```

#### `geo::destination(origin, bearing, third[, extras])`

The third argument selects how far to travel, dispatched on type:

- **Number**: distance in meters
- **Duration capsule**: travel for that duration at `origin.speed` (m/s)
- **Time capsule**: travel until that time at `origin.speed`

`extras` is an optional object merged onto `origin` before the calculation
(extras win on key conflicts). All fields from the effective origin are
preserved in the result; only `lat`/`lon` (and `time` for duration/time forms)
are updated.

```hcl
dest = geo::destination(var.location, 90.0, 1000.0)
future = geo::destination(var.vehicle, var.vehicle.track, duration("5m"))
projected = geo::destination({lat = 37.7, lon = -122.4}, 90.0, duration("10m"),
                            {speed = 15.0, time = now()})
```

#### `geo::waypoints(point_a, point_b, n)`

Returns a list of `n` (>= 2) evenly spaced `{lat, lon, track}` objects along
the geodesic, including both endpoints.

```hcl
waypoints = geo::waypoints(var.origin, var.destination, 5)
```

---

### Geometric Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `geo::area` | `geo::area(polygon)` | Area in square meters |
| `geo::contains` | `geo::contains(polygon, point)` | Point-in-polygon test |
| `geo::nearest` | `geo::nearest(polygon, point)` | Nearest point on perimeter |
| `geo::line_intersect` | `geo::line_intersect(line_a, line_b)` | Intersection points of two polylines |

A polygon is a `list(point)`, implicitly closed. Fewer than 3 points is an error.

#### `geo::area(polygon)`

Returns area in square meters. Orientation-independent.

#### `geo::contains(polygon, point)`

Returns `true` if `point` is inside the polygon.

```hcl
geo::contains(var.home_zone, var.vehicle_position)
```

#### `geo::nearest(polygon, point)`

Returns the closest `{lat, lon}` on the polygon perimeter (may be mid-edge).

#### `geo::line_intersect(line_a, line_b)`

Returns a list of `{lat, lon}` intersection points. Empty list if no crossings.
Each line must have at least 2 points.

---

## Dependencies

| Functionality | Library |
|---------------|---------|
| Sunrise/sunset | [go-sunrise](https://github.com/nathan-osman/go-sunrise) |
| Sun/moon position, moon phase | [suncalc](https://github.com/kixorz/suncalc) |
| Geodesic (WGS-84) | [geographiclib-go](https://github.com/natemcintosh/geographiclib-go) |
| Geometry (area, contains, etc.) | [S2 Geometry](https://github.com/golang/geo) |

## Notes

- All distances are in meters; all bearings and angles are in degrees.
- Solar functions return time capsules from [time-cty-funcs](https://github.com/tsarna/time-cty-funcs) and accept duration/time capsules as arguments.
- Geodesic functions use the WGS-84 ellipsoid model for high accuracy.
