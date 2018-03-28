dp-map-renderer
================

Renders an SVG representation of a choropleth map given a topojson file and data.

See the [exampleResponse](testdata/exampleResponse.html) for an illustration of the sort of map that can be generated.
The testdata folder also includes example requests for both the render and analyse endpoints.


| Environment variable       | Default                  | Description                                            |
| -------------------------- | ------------------------ | -----------                                            |
| BIND_ADDR                  | :23500                   | The host and port to bind to                           |
| CORS_ALLOWED_ORIGINS       | *                        | The allowed origins for CORS requests                  |
| SHUTDOWN_TIMEOUT           | 5s                       | The graceful shutdown timeout ([`time.Duration`](https://golang.org/pkg/time/#Duration) format) |

### Endpoints

| url                   | Method | Parameter values             | Description                                                                                                                                                                                                                                       |
| ---                   | ------ | ----------------             | -----------                                                                                                                                                                                                                                       |
| /render/{render_type} | POST   | render_type = `svg` or `png` | Renders the (json) data provided in the post body as an html figure with either an svg or png map                                                                                                                                                    |
| /analyse              | POST   |                              | Accepts json containing a topojson topology and a string representation of a csv file. Validates that the csv file matches the topology and returns the contents of the csv in json format. Also identifies natural breaks for the choropleth map |

### Healthchecking

Currently reported on endpoint `/healthcheck`. There are no other services consumed, so it will always return OK.

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2018, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.

Some of the transformation of geojson to svg is based on original work by Fabian Prein: [geojson2svg](https://github.com/fapian/geojson2svg)