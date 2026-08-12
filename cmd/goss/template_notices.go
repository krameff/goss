package main

import "log/slog"

// sprout (the template function library behind gossfile templating) logs a
// slog warning every time a template calls one of the legacy sprig function
// names -- `upper`, `get`, `toYaml` and friends. Those names are still fully
// supported here, and some appear in goss's own documented examples, so the
// warnings are noise rather than something a user running `goss validate` can
// act on. sprout gives no way to inject a logger on its backward-compatible
// handler: it reads slog.Default() at call time.
//
// goss logs through the stdlib log package, never slog, so raising the level
// of slog's default log bridge silences those notices and nothing else.
// slog.SetDefault is deliberately not used here -- it would also reroute the
// log package's own output through slog and reformat every existing goss
// message.
func silenceTemplateDeprecationNotices() {
	slog.SetLogLoggerLevel(slog.LevelError)
}
