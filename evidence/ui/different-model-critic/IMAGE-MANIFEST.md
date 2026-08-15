# Sanitized critic image manifest

The critic receives exactly these seven copied PNGs and the neutral rubric in `PROMPT.md`. No bearer token, URL fragment, private repository path, captured body, implementation commentary, source file, or prior finding is included.

| critic filename | neutral state label | viewport | bytes | SHA-256 |
|---|---|---:|---:|---|
| `01-decision-ready-desktop.png` | decision-ready | 1440×900 | 164235 | `eb91fecebc5fd2a574a84c2df960c923b2d2984c898ed0f38135afa8c17bb3f0` |
| `02-decision-ready-mobile.png` | decision-ready | 375×812 | 64892 | `3cfc8737b8daff9b5e97a339c3264f6d11892c493887eed76f863e795d7b326e` |
| `03-error-mobile.png` | error | 375×812 | 55751 | `093c053961b5cef2bce16a9f661ad1af796c93764e091ec6fcd37fc9d1f7dfd7` |
| `04-identity-reveal-desktop.png` | identity-reveal | 1440×900 | 164198 | `b8a7bd48f68bc19e1d7706e4b5de7b36efca88a7d7bedbc70e1fee77e3baea44` |
| `05-resolved-desktop.png` | resolved | 1440×900 | 148199 | `17af68b65f9eace25366ecfbbbbb2e3f71e64cee788d518b607215eb48f6c5cf` |
| `06-stale-mobile.png` | stale | 375×812 | 56387 | `1c4333752b06101ec963ddc30e38de4f816186776cd0c9943edf801367a0209a` |
| `07-deferred-mobile.png` | deferred | 375×812 | 60703 | `0e80db14f2cbe666b945c2765125661f04af01fa83475c5106073251a5ba84da` |

The external model's qualitative judgment remains `UNRECEIPTED` and semantically inert. The didrun record establishes only that the exact CLI observation ran and exited; it does not establish taste, comprehension, correctness, accessibility, or security.
