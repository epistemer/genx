# Input: every labels/v1 file of the app, as one array, ordered by file path.
# Output: one unit per package.
def norm: {key, text, path: (.path // ""), var: (.var // "")};
def twice(f): group_by(f) | map(select(length > 1) | .[0] | f);

# One key has one text across the app.
([.[] | .producer as $p | .labels[] | {key, text, producer: $p}]
   | group_by(.key)
   | map(select((map(.text) | unique | length) > 1)
         | "\(.[0].key) (\(map(.producer) | unique | join(", ")))")) as $clash
| if ($clash | length) > 0
  then error("labels: one key, two texts: " + ($clash | join("; ")))
  else . end
| group_by(.package.dir)[]
| { package: .[0].package,
    out: "\(.[0].package.dir)/zz_\(.[0].package.name)_i18n.go",
    labels: [.[].labels[] | norm] }
# In a package, a variable or a path names one label.
| ([.labels[] | select(.var != "")] | twice(.var)) as $vars
| ([.labels[] | select(.path != "")] | twice(.path)) as $paths
| if ($vars + $paths | length) > 0
  then error("labels: declared twice in \(.package.dir): " + ($vars + $paths | join(", ")))
  else . end
| .vars   = [.labels[] | select(.var != "")]
| .hidden = [.labels[] | select(.var == "" and .path == "")]
| .paths  = [.labels[] | select(.path != "")]
