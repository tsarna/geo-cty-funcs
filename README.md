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

## Functions

### Point Construction and Formatting

| Function | Signature | Description |
|----------|-----------|-------------|
| `geo_point` | `geo_point(combined)` or `geo_point(lat, lon[, base])` | Constructs a point from a combined string or separate lat/lon |
| `geo_format` | `geo_format(point[, format])` | Formats a point as a string |

#### `geo_point(combined[, base])` or `geo_point(lat, lon[, base])`

A single combined string is parsed by splitting on `,` (decimal formats) or on
the hemisphere-letter boundary (DMS formats). All `geo_format` output formats
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
geo_point("37.7749,-122.4194")
geo_point("37°46'29\"N 122°25'9\"W")
geo_point(37.7749, -122.4194)
geo_point("37°46'29\"N", "122°25'9\"W")
geo_point(37.7749, -122.4194, {alt = 11.0, track = 90.0})
geo_point("37.7749,-122.4194", {alt = 11.0})
```

#### `geo_format(point[, format])`

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
occurs again. `solar_noon` and `solar_midnight` require both sunrise and
sunset to exist on the relevant day(s); during polar day or polar night they
return the first meaningful occurrence after polar conditions end.

| Function | Description |
|----------|-------------|
| `sunrise` | Next sunrise |
| `sunset` | Next sunset |
| `solar_noon` | Next solar noon (midpoint of sunrise and sunset) |
| `solar_midnight` | Next solar midnight (midpoint of sunset and next sunrise) |

```hcl
time = sunrise(var.location)
time = sunrise(var.location, duration("-30m"))
time = sunset(var.location, duration("-15m"), ctx.scheduled_time)
```

---

### Sun and Moon Position

| Function | Signature | Returns |
|----------|-----------|---------|
| `sun_position` | `sun_position(point[, t])` | `{azimuth, altitude}` (degrees) |
| `moon_position` | `moon_position(point[, t])` | `{azimuth, altitude, distance}` (degrees, meters) |
| `moon_phase` | `moon_phase([t])` | `{fraction, phase, angle}` |

- **azimuth**: clockwise from true north (0=N, 90=E, 180=S, 270=W)
- **altitude**: degrees above horizon (negative = below)
- **distance**: meters to the moon's centre
- **fraction**: 0-1, illuminated fraction of the visible disc
- **phase**: 0-1, position in lunation cycle (0 = new, 0.5 = full)
- **angle**: bright-limb angle in degrees (sign distinguishes waxing/waning)

```hcl
s = sun_position(var.location)
s.altitude > 0            # true if the sun is up
s.altitude < -6.0         # true if past civil twilight

mp = moon_phase()
mp.phase < 0.5            # true if waxing
```

---

### Geodesic Functions (WGS-84)

| Function | Signature | Description |
|----------|-----------|-------------|
| `geo_inverse` | `geo_inverse(a, b)` | Distance and bearings between two points |
| `geo_destination` | `geo_destination(origin, bearing, third[, extras])` | Point reached by travelling on a bearing |
| `geo_waypoints` | `geo_waypoints(a, b, n)` | n evenly spaced points along the geodesic |

#### `geo_inverse(point_a, point_b)`

Returns `{distance, bearing, back_bearing}`. Distance in meters, bearings in
degrees clockwise from north.

```hcl
r = geo_inverse(var.current, var.destination)
r.distance    # meters
r.bearing     # heading toward destination
```

#### `geo_destination(origin, bearing, third[, extras])`

The third argument selects how far to travel, dispatched on type:

- **Number**: distance in meters
- **Duration capsule**: travel for that duration at `origin.speed` (m/s)
- **Time capsule**: travel until that time at `origin.speed`

`extras` is an optional object merged onto `origin` before the calculation
(extras win on key conflicts). All fields from the effective origin are
preserved in the result; only `lat`/`lon` (and `time` for duration/time forms)
are updated.

```hcl
dest = geo_destination(var.location, 90.0, 1000.0)
future = geo_destination(var.vehicle, var.vehicle.track, duration("5m"))
projected = geo_destination({lat = 37.7, lon = -122.4}, 90.0, duration("10m"),
                            {speed = 15.0, time = now()})
```

#### `geo_waypoints(point_a, point_b, n)`

Returns a list of `n` (>= 2) evenly spaced `{lat, lon, track}` objects along
the geodesic, including both endpoints.

```hcl
waypoints = geo_waypoints(var.origin, var.destination, 5)
```

---

### Geometric Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `geo_area` | `geo_area(polygon)` | Area in square meters |
| `geo_contains` | `geo_contains(polygon, point)` | Point-in-polygon test |
| `geo_nearest` | `geo_nearest(polygon, point)` | Nearest point on perimeter |
| `geo_line_intersect` | `geo_line_intersect(line_a, line_b)` | Intersection points of two polylines |

A polygon is a `list(point)`, implicitly closed. Fewer than 3 points is an error.

#### `geo_area(polygon)`

Returns area in square meters. Orientation-independent.

#### `geo_contains(polygon, point)`

Returns `true` if `point` is inside the polygon.

```hcl
geo_contains(var.home_zone, var.vehicle_position)
```

#### `geo_nearest(polygon, point)`

Returns the closest `{lat, lon}` on the polygon perimeter (may be mid-edge).

#### `geo_line_intersect(line_a, line_b)`

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
