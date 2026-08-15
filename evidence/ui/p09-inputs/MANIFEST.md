# P09 sanitized screenshot inputs

These images are deterministic inputs to the separately sealed P10 visual
build loop. They were captured from the real embedded Countershape binary with
Chromium in dark mode and reduced motion. Screenshot existence is not a visual
quality claim.

| File | Viewport / rendered size | Bytes | SHA-256 |
| --- | --- | ---: | --- |
| `decision-ready-desktop.png` | 1440×900 / 1440×1732 | 536672 | `800dad8dbe62bfb2242928dbe267fbf9f38f0fcec66ab9051343762fa8e8a8e3` |
| `decision-ready-mobile.png` | 375×812 / 375×3492 | 408960 | `be77a2dfde47f3093550be869d537e21ce9347cc622db19becd19ce37380d11e` |
| `error-mobile.png` | 375×812 / 375×1208 | 138486 | `c27fd0922e22b23e32ef3d1612c7c2c991bb9207f4a03156e0b19dde28312ee3` |
| `identity-reveal-desktop.png` | 1440×900 / 1440×2392 | 772123 | `8f2a116a6e139b27bd72d791acec23f073b43a6b6d58e45591785fce96f71770` |
| `resolved-desktop.png` | 1440×900 / 1440×1971 | 525434 | `ff01cbd75acfee3bad35cfa6b6a51e1db1550638c283bdc3bf8effc39f37beb5` |

The capture set contains only deterministic seeded fixture labels and package
facts. It contains no bearer token, URL fragment, user repository path,
credential, captured user body, or user-authored private data. A second fresh
capture was byte-identical for every PNG.
