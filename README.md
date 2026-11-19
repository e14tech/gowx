# gowx

gowx prints the National Weather Service forecast for a US ZIP code.

## Usage

From this directory:

```
go run . 50583
go run . -zip 50583
go run . -hourly 50583
```

With no ZIP, gowx asks for one. Press Ctrl-C to cancel a request that is still running.

`-hourly` prints the hourly forecast instead of the daily summary. It is off by default. Each hour is one line: time, temperature, wind, and the short forecast. The hourly feed usually leaves `name` blank, so gowx labels the row with the start time (`Thu 11 AM`).

## Output

The default view has three parts:

1. **TODAY** — temperature, wind, and the full `detailedForecast` for the current period. When the forecast starts at night (a normal evening run), this is tonight's low and description.
2. **NEXT 7 DAYS** — one line per day: the day name, the daytime high, the following night's low, and the short forecast (`Sunny`, `Mostly Cloudy`, and so on). At most 7 days. Today's line is included so the overnight low sits next to the high. A leading night period is not a row in this list.
3. **ALERTS** — active alerts for that point, or `none`.

## How it works

Three HTTP calls, in order:

1. **ZIP to latitude and longitude.** `GET https://nominatim.openstreetmap.org/search` with `postalcode`, `country` (`United States`), and `format=json`. The body is a JSON array. gowx uses the first hit and rounds the coordinates to 4 decimal places.
2. **Coordinates to forecast links.** `GET https://api.weather.gov/points/{lat},{lon}` returns `properties.forecast`, `properties.forecastHourly`, and the city and state.
3. **Forecast and alerts.** gowx GETs the forecast URL from step 2 (or the hourly URL with `-hourly`) and `https://api.weather.gov/alerts/active?point={lat},{lon}`.

Nominatim's usage policy asks for a descriptive User-Agent and light traffic. gowx sends `gowx (github.com/e14tech/gowx)` and makes one geocode request per run. NWS wants that same User-Agent plus `Accept: application/geo+json`.

One shared `http.Client` has a timeout. Network errors and HTTP 5xx responses are retried with exponential backoff. The retry stops if you hit Ctrl-C.

Query strings are built with `net/url`, not `fmt.Sprintf`, so characters like the space in `United States` are escaped by the URL package.

## Example

`go run . 50583` (Sac City, IA):

```
Sac City, IA (42.4171, -94.9960)

TODAY
Today: high 64°F, wind 2 to 10 mph SW
Sunny. High near 64, with temperatures falling to around 57 in the afternoon. Southwest wind 2 to 10 mph.

NEXT 7 DAYS
Today            high 64°F, low 44°F, Sunny
Friday           high 67°F, low 52°F, Mostly Sunny
Saturday         high 59°F, low 32°F, Rain Showers
Sunday           high 43°F, low 20°F, Sunny
Monday           high 40°F, low 19°F, Mostly Sunny
Tuesday          high 36°F, low 18°F, Partly Sunny
Wednesday        high 37°F, low 18°F, Mostly Sunny

ALERTS
none
```
