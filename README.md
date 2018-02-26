dp-map-renderer
================

Renders an SVG representation of a choropleth map given a topojson file and data.


| Environment variable       | Default                  | Description                                            |
| -------------------------- | ------------------------ | -----------                                            |
| BIND_ADDR                  | :23500                   | The host and port to bind to                           |
| CORS_ALLOWED_ORIGINS       | *                        | The allowed origins for CORS requests                  |
| SHUTDOWN_TIMEOUT           | 5s                       | The graceful shutdown timeout ([`time.Duration`](https://golang.org/pkg/time/#Duration) format) |

### Endpoints

| url                   | Method | Parameter values     | Description                                                                                   |
| ---                   | ------ | ----------------     | -----------                                                                                   |
| /render/{render_type} | POST   | render_type = `html` | Renders the (json) data provided in the post body as an html figure with svg content          |

### Healthchecking

Currently reported on endpoint `/healthcheck`. There are no other services consumed, so it will always return OK.

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2018, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.
