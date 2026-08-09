#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { lstat, open, readdir, readFile, realpath } from "node:fs/promises";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

const modulePath = fileURLToPath(import.meta.url);
export const repositoryRoot = resolve(dirname(modulePath), "..");
const specificationPath = resolve(repositoryRoot, "spec/verification/u7-unit-paths.json");
const modulePrefix = "github.com/nelsonwerd/countershape/";
const phases = Object.freeze(["U7P", "U7A", "U7B", "U7C", "U7D"]);

const governedRoots = Object.freeze([
	"cmd/countershape",
	"internal/reference",
	"testkit/reference",
]);
const governedDriverPattern = /^tools\/run-u7-[a-z0-9-]+\.mjs$/u;
const knownDrivers = Object.freeze([
	"tools/run-u7-http-study.mjs",
	"tools/run-u7-cli-study.mjs",
]);

const declaredSurfaceByPhase = Object.freeze({
	U7P: Object.freeze([]),
	U7A: Object.freeze([
		"cmd/countershape/main.go",
		"internal/reference/app/command.go",
		"internal/reference/app/envelope.go",
		"internal/reference/app/render.go",
		"internal/reference/app/workspace.go",
		"internal/reference/app/spec.go",
		"internal/reference/app/validate.go",
		"internal/reference/app/preflight.go",
		"internal/reference/app/app_test.go",
		"internal/reference/app/cli_e2e_test.go",
	]),
	U7B: Object.freeze([
		"cmd/countershape/main.go",
		"internal/reference/app/command.go",
		"internal/reference/app/envelope.go",
		"internal/reference/app/render.go",
		"internal/reference/app/workspace.go",
		"internal/reference/app/run_darwin.go",
		"internal/reference/httpstudy/study.go",
		"internal/reference/httpstudy/reduction.go",
		"internal/reference/httpstudy/result.go",
		"internal/reference/httpstudy/study_test.go",
		"internal/reference/httpstudy/e2e_darwin_test.go",
		"testkit/reference/http.go",
		"testkit/reference/http_test.go",
		"tools/run-u7-http-study.mjs",
	]),
	U7C: Object.freeze([
		"cmd/countershape/main.go",
		"internal/reference/app/command.go",
		"internal/reference/app/envelope.go",
		"internal/reference/app/render.go",
		"internal/reference/app/workspace.go",
		"internal/reference/clistudy/study.go",
		"internal/reference/clistudy/reduction.go",
		"internal/reference/clistudy/choice.go",
		"internal/reference/clistudy/contract.go",
		"internal/reference/clistudy/inspect.go",
		"internal/reference/clistudy/study_test.go",
		"internal/reference/clistudy/choice_test.go",
		"internal/reference/clistudy/contract_darwin_test.go",
		"testkit/reference/cli.go",
		"testkit/reference/cli_test.go",
		"tools/run-u7-cli-study.mjs",
	]),
	U7D: Object.freeze([
		"cmd/countershape/main.go",
		"internal/reference/app/command.go",
		"internal/reference/app/envelope.go",
		"internal/reference/app/render.go",
		"internal/reference/app/workspace.go",
		"internal/reference/reproduce/run_darwin.go",
		"internal/reference/reproduce/run_darwin_test.go",
		"internal/reference/app/architecture_test.go",
		"internal/reference/app/exitcodes_test.go",
	]),
});

const phaseIdentity = Object.freeze({
	U7P: Object.freeze({ parent: "C6B", profile: "SOURCE_FULL", productAuthority: "NONE", productBehavior: "INHERITED_UNREPROVEN", receipt: "ABSENT" }),
	U7A: Object.freeze({ parent: "U7P", profile: "SOURCE_FULL", productAuthority: "U7_REFERENCE_APPLICATION", productBehavior: "CANDIDATE_UNRECEIPTED", receipt: "ABSENT" }),
	U7B: Object.freeze({ parent: "U7A", profile: "SOURCE_FULL", productAuthority: "U7_REFERENCE_APPLICATION", productBehavior: "CANDIDATE_UNRECEIPTED", receipt: "ABSENT" }),
	U7C: Object.freeze({ parent: "U7B", profile: "SOURCE_FULL", productAuthority: "U7_REFERENCE_APPLICATION", productBehavior: "CANDIDATE_UNRECEIPTED", receipt: "ABSENT" }),
	U7D: Object.freeze({ parent: "U7C", profile: "SOURCE_FULL", productAuthority: "U7_REFERENCE_APPLICATION", productBehavior: "CANDIDATE_UNRECEIPTED", receipt: "ABSENT" }),
});

const protectedRoots = Object.freeze([
	"internal/canon",
	"internal/domain",
	"internal/observe",
	"internal/compare",
	"internal/reduce",
	"internal/reduction",
	"internal/choice",
]);
const protectedToolPaths = Object.freeze([
	"tools/check-u5-architecture.mjs",
	"tools/check-u5-architecture-selftest.mjs",
	"tools/check-u6-architecture.mjs",
	"tools/check-u6-architecture-selftest.mjs",
]);
const protectedRows = Object.freeze([
	Object.freeze({"path":"internal/canon/canon_test.go","mode":"100644","type":"blob","blob":"103ec9962c786b6014aa89365b8d881f599bb591","bytes":21882,"sha256":"6a0301d03741afb8659f76839da205f3d700c0bfac0839c4b59d0afa2d8e5b62"}),
	Object.freeze({"path":"internal/canon/digest.go","mode":"100644","type":"blob","blob":"99b6bfa047e75dba02ebf69b8c236e0d881e8378","bytes":1952,"sha256":"1362640008623e143e799d88b66151dcb9e4032b4524dbe6d3f4218890c874cc"}),
	Object.freeze({"path":"internal/canon/doc.go","mode":"100644","type":"blob","blob":"0a9bcf9a04ab6e95c7cf9fbc6bb7cd7f69891941","bytes":604,"sha256":"bc3f8f25e97c6ab253c11e8f6a866a6e343151da5d123b7c2f29de51e61044e8"}),
	Object.freeze({"path":"internal/canon/encode.go","mode":"100644","type":"blob","blob":"c86546ca0cb6df73e9434a84d494059f274fc1cf","bytes":4767,"sha256":"785e8f2f455b1a2bb54d7e09c5d084149294ea171ccab2e4dc858e2259c3f112"}),
	Object.freeze({"path":"internal/canon/error.go","mode":"100644","type":"blob","blob":"405b9cb318fdfcbc276d16711a1133b942db8064","bytes":2294,"sha256":"abe24ffcdfde8493f119c27b5a0b1654ca985cbcf1de5cf9178603c8afdd756f"}),
	Object.freeze({"path":"internal/canon/fuzz_test.go","mode":"100644","type":"blob","blob":"f92c29dc43afb358e99cded413f29581e0c87886","bytes":1083,"sha256":"7e18e589eded683500f1a334022aab3ce2ea132c6a09aae9f11da106ad7af698"}),
	Object.freeze({"path":"internal/canon/limits.go","mode":"100644","type":"blob","blob":"24d9c417ed1ae9cbfd9bcaa3786235e245ceb212","bytes":1190,"sha256":"a55daa13f00ef2b0a7882fad28326648926cbc0a025a2c95ceb7c7dc98f98be5"}),
	Object.freeze({"path":"internal/canon/limits_test.go","mode":"100644","type":"blob","blob":"a58350e2f99fa635679c4fc0d31b28a7ba196b77","bytes":4265,"sha256":"74ec1d4e80b1a41cfad218868eddbe00400985d40e101c54fae64f032e8f5303"}),
	Object.freeze({"path":"internal/canon/parse.go","mode":"100644","type":"blob","blob":"55009f7774986db7690db329977ec6897bc62f18","bytes":6809,"sha256":"d606d106be5d3dbb1ed6c145c34f09b41d444fdbbf5252af3e5885368d69eefb"}),
	Object.freeze({"path":"internal/canon/scan.go","mode":"100644","type":"blob","blob":"e87272a361db9d909efdc0cb009389018acac14b","bytes":7939,"sha256":"44385489a23fc6132e77a8960bc758da5ff23afb07e5608d22ba5a2e1db5b679"}),
	Object.freeze({"path":"internal/canon/token.go","mode":"100644","type":"blob","blob":"47b167d29eabcdd641919c88b434e7675af1aac2","bytes":1216,"sha256":"f75e8bb64a8d74f50e5dd2ba5641b69fb5bf2848b357d031bb5cd1db4cfedb85"}),
	Object.freeze({"path":"internal/canon/typed.go","mode":"100644","type":"blob","blob":"ceb188ed8834d2fc7de24a877b9e08223daa0c0e","bytes":9788,"sha256":"a500459904bd3164f96c73cb90e6dcb7c5966c6e9e57e728b33d213e84d9047a"}),
	Object.freeze({"path":"internal/canon/value.go","mode":"100644","type":"blob","blob":"4a4f86114af7b5a09f1f0af52214f07b85a2dd90","bytes":5465,"sha256":"a8b0438e92adfbf56bc4128da909e541664bf0b0e76e00d03512c89c03662bbf"}),
	Object.freeze({"path":"internal/choice/blind.go","mode":"100644","type":"blob","blob":"27afd14c787af183d979ff480d07e97fc0a547f8","bytes":15203,"sha256":"052ca1b0a0953f35b68349adfe8f1337c363747520ea3b5c230f7d3021fcc2bf"}),
	Object.freeze({"path":"internal/choice/blind_test.go","mode":"100644","type":"blob","blob":"cfacd18ffb20bd46be08f5d5d54aaa2e00f298fb","bytes":11960,"sha256":"8d8afae8ec897998548359b0866dee030795b1274091aba13447fd2681857bbf"}),
	Object.freeze({"path":"internal/choice/choicepoint.go","mode":"100644","type":"blob","blob":"6f137220cfda9af889556ee19c82a176eaad536a","bytes":29104,"sha256":"ff63d0c86eaa421d3a53a977a1684a00b7b9803001b8a850904328698d6b6791"}),
	Object.freeze({"path":"internal/choice/decision.go","mode":"100644","type":"blob","blob":"bf977d7fddef70e73938805441c7a799e93a6591","bytes":26549,"sha256":"c26e3ec97f44d652b97b20a58114249796cf79360fe6d20ee748607d756f16ca"}),
	Object.freeze({"path":"internal/choice/legacy_validation_fixtures_test.go","mode":"100644","type":"blob","blob":"25a24606be85638dd3810e0798de09e98fedb114","bytes":3279,"sha256":"b43c6dfa0f4d9e6fe61f258e51662c006ac2dc710cd1082ea973e50936153c87"}),
	Object.freeze({"path":"internal/choice/portable.go","mode":"100644","type":"blob","blob":"8d0ddca3e1ea557e3619ff6b0b650669091098ae","bytes":12670,"sha256":"1ffc984a33c03e564bd7547fd8bc76442aff081152d600d0923124184662745a"}),
	Object.freeze({"path":"internal/choice/portable_choice_test.go","mode":"100644","type":"blob","blob":"a416260627aa99db0e61a2dc8b943b697dbf4759","bytes":27671,"sha256":"bada61a2cb5a71f57ee2989dfa3bc671de7d03aaea66f1afdabde05a3d1d6507"}),
	Object.freeze({"path":"internal/choice/projectiontranslate_test.go","mode":"100644","type":"blob","blob":"2370b49b19d2cab0ac19b1fc9e8eae9880af8337","bytes":7417,"sha256":"974608cbbb3a7f171bdb0f7003cfa33629b4ccc04fef21acbd0ca3d74358f660"}),
	Object.freeze({"path":"internal/choice/promotion/authority/authority.go","mode":"100644","type":"blob","blob":"4841814559782fd9cdbaff41f8d09f3254989f8a","bytes":316,"sha256":"00235af653263eeeb4dfb8fa1d6f0573a1955dc354b1ae4ac9dc0b9a45a6eb04"}),
	Object.freeze({"path":"internal/choice/promotion/internal/publication/authority.go","mode":"100644","type":"blob","blob":"a22ed558cf8f62264a299c04f0fae1dbb3116542","bytes":4346,"sha256":"8239dd39c41f572a78a71ee24a208054629434ae32af20c5ac9e2dfbe1d2b43f"}),
	Object.freeze({"path":"internal/choice/promotion/internal/publication/authority_test.go","mode":"100644","type":"blob","blob":"acb6d4f438ec6105eaaa28b3348a2b9e5c8d9c35","bytes":7798,"sha256":"073056904b2bc6b0af8e296949710ede6dd8d2d5d77d1e10ccb332d389cd90b9"}),
	Object.freeze({"path":"internal/choice/promotion/legacy_snapshot_darwin_test.go","mode":"100644","type":"blob","blob":"926f68a3b70d9b0a77be602ed893db490d2b1c3c","bytes":10127,"sha256":"68b283a3a0b7d519b1584c99868d86f52718f393cb19bb1641284b346f5c26f8"}),
	Object.freeze({"path":"internal/choice/promotion/residue.go","mode":"100644","type":"blob","blob":"d21b397ee05f9d1632c4314b174f946a4a4c234f","bytes":6155,"sha256":"0ad60120b0add6e11002939c985f19e45494e72ec012534ef9df982233f29aa4"}),
	Object.freeze({"path":"internal/choice/promotion/service.go","mode":"100644","type":"blob","blob":"b1a8edfce475b93db59935bc67c6642c8339c281","bytes":32106,"sha256":"86b5d2f78422c4882b9679d76b4a98cabd306c4a9449d9d4af1dc014c9990ad7"}),
	Object.freeze({"path":"internal/choice/schema_parity_test.go","mode":"100644","type":"blob","blob":"09034a2f5ef12bf7b2e49b70a28648e293b0b362","bytes":5229,"sha256":"ca9ba1440fded731dc8756b92fa3c7ef68c71097487b2d52e757a4e87161c695"}),
	Object.freeze({"path":"internal/choice/session.go","mode":"100644","type":"blob","blob":"62dd245dd9c35bbf350539a65d7e08d6e91229d1","bytes":24502,"sha256":"962b67cabe5e533f83eb434da62de7a29cc9d03ba9831e69798987e0f4fce609"}),
	Object.freeze({"path":"internal/choice/session_roundtrip_test.go","mode":"100644","type":"blob","blob":"d89e459a08bfca717f6fb18b3d82e2f46ea483fa","bytes":31015,"sha256":"e7abce945ce9f2bdfc2451d3478bdbb54369b03fe66470ce56cf96dddde7088e"}),
	Object.freeze({"path":"internal/choice/validation.go","mode":"100644","type":"blob","blob":"e62baecad054a2967de693e5b0ed45fe239c5171","bytes":65906,"sha256":"3e3a5f0792d872bdd34f4d45f6f8677c69001f09db554c4bf66a32739842db51"}),
	Object.freeze({"path":"internal/choice/validation_test.go","mode":"100644","type":"blob","blob":"87fcbaa853cabb6670568fadfc24ad05ae0d805e","bytes":58919,"sha256":"174ee5bd778bb199acbdfb2660ce59a6e38c86c481612188d94e4a3712f71c85"}),
	Object.freeze({"path":"internal/compare/outcome_map.go","mode":"100644","type":"blob","blob":"c7d3f32a06253e3c13e5a1204ed29d679d0c1851","bytes":33934,"sha256":"e6cde7c3689bba5696277037811a9bf7b445125f9570676129cfcbc7f2356b8a"}),
	Object.freeze({"path":"internal/compare/outcome_map_test.go","mode":"100644","type":"blob","blob":"e863170754c5a707ce994ee3e500a971ef50049b","bytes":33435,"sha256":"17c93acf30d53c66af5522f534947e8a3a7a662216e535b27d3904bf348075b2"}),
	Object.freeze({"path":"internal/compare/wire.go","mode":"100644","type":"blob","blob":"0fe92f7c49614d6a9e06b332c07e6e5a528f691c","bytes":13099,"sha256":"baf7f6fab5228919295562e4efb349f773bc669b21be38a5e3db50830c5f0a60"}),
	Object.freeze({"path":"internal/domain/domain_test.go","mode":"100644","type":"blob","blob":"1798787d58f747fe559e9f49190279497bd89394","bytes":23527,"sha256":"27ff69cc148d3ece3111b745be72231baa7a315e55bb71d36535a56ee686f4b9"}),
	Object.freeze({"path":"internal/domain/envelope.go","mode":"100644","type":"blob","blob":"9ca44f8c5a7b92344e2bc516cb4ca621d4cef7b6","bytes":36973,"sha256":"8d94b966d5a2e82e345d493c925c3e0e2899362746220f25e8a1aed5ff4010d2"}),
	Object.freeze({"path":"internal/domain/errors.go","mode":"100644","type":"blob","blob":"7974189764ba838d2c37cb0324a93c0e251f925a","bytes":1190,"sha256":"ef7990b604b74730a4495f63f6de4aaaa81bc5bbb0c97008251baffa27837d1c"}),
	Object.freeze({"path":"internal/domain/identity.go","mode":"100644","type":"blob","blob":"c5434dc402ec36e4ee885950496abb7d4708d162","bytes":7003,"sha256":"41903bfe435728978ca7713b283b70839c9f3e415ecd3aec2ecc7e74049400a8"}),
	Object.freeze({"path":"internal/domain/projection.go","mode":"100644","type":"blob","blob":"5352903e2239815aaa76b4d4801cd795eb221ef7","bytes":9657,"sha256":"d281a67e2cd2c348f683083965637ffc1f5b71da4ce53de7fb25e9a80bc3dc84"}),
	Object.freeze({"path":"internal/domain/receipt.go","mode":"100644","type":"blob","blob":"8ddfca0ad2d02c07027dc9f55504ca080d1c86e6","bytes":1819,"sha256":"78a4713137677de556b0c996e42d7f5616c6e9483207b78e443efda5b76ff0a9"}),
	Object.freeze({"path":"internal/domain/transitions.go","mode":"100644","type":"blob","blob":"1e6a129474be5834ecc1ad94bc7b9fa99be99a2c","bytes":8154,"sha256":"978240956cd065c5fccd924606d3fd8cfc7a47baee0a65bdba05594f5ee2cbaa"}),
	Object.freeze({"path":"internal/domain/world.go","mode":"100644","type":"blob","blob":"d430d4d97ab5db9b81c127c4aeafef4ec8d9899e","bytes":30563,"sha256":"5c83df7c28163fb78ba7c01fe7099a0b44e93dae1cc50dc760b332fec17ee735"}),
	Object.freeze({"path":"internal/observe/batch.go","mode":"100644","type":"blob","blob":"ae27a681c1a208f10706612e8fcfffdbb128d8ed","bytes":19554,"sha256":"104c9c3f77042e4cdf2befd760510f83019dd0ffddce573af766a4e9a854aceb"}),
	Object.freeze({"path":"internal/observe/batch_test.go","mode":"100644","type":"blob","blob":"50474c4857cea459c5b8f50a69a5cb760b81903c","bytes":24109,"sha256":"2cc090494ee1f8f52eb22c34b3ddfd02f8b725db93be14ed18bffc9a59bd3e1a"}),
	Object.freeze({"path":"internal/observe/eligibility.go","mode":"100644","type":"blob","blob":"3f02d183e8cb68b509c3373626957a7f6495f9cc","bytes":30351,"sha256":"3a2f840b92ff7b88d1046c5162f618fb60fc9d7b536b22d0342fa810dc1e44a8"}),
	Object.freeze({"path":"internal/observe/eligibility_test.go","mode":"100644","type":"blob","blob":"095d0f1d7b25b736181d0aa7e4ce8bf57fc4803b","bytes":34797,"sha256":"32aed5ad5950f958d1a0a8342dc3f78afb1e1ad5143bb14b229632305ac4d3de"}),
	Object.freeze({"path":"internal/observe/eligibilitycore/eligibility.go","mode":"100644","type":"blob","blob":"1948bcfc3bcc07e8ea2f3b68cd2d7f00a28d23c9","bytes":2785,"sha256":"237d28d224c961aac8e1fe0bcca5ae2de0e0828aa6b393c786260f3da735b0d3"}),
	Object.freeze({"path":"internal/observe/executed_link.go","mode":"100644","type":"blob","blob":"c7c5852378e9079ac041ea2c0d3f700443562ddd","bytes":3265,"sha256":"c4ef18f7da79e2e3541935fe54bac93250e6f51085095bc4001c1c66b7a3cfe4"}),
	Object.freeze({"path":"internal/observe/identity.go","mode":"100644","type":"blob","blob":"8a6574253a3119544c23b2c3d55eba1512aa0452","bytes":6279,"sha256":"57009acd70390d86e88897c82266df9b5dff10f813b44aa480243a8b51bd0b21"}),
	Object.freeze({"path":"internal/observe/projection_evidence.go","mode":"100644","type":"blob","blob":"22a78bdeda03f8c19e0ddf43579381944d57bd94","bytes":10753,"sha256":"7c3fb297480112e59570cd9bb447213497cc3c7ec3c121f8f38018c6152f92d3"}),
	Object.freeze({"path":"internal/observe/schedule.go","mode":"100644","type":"blob","blob":"3908d5a59abe46328c4fa4e320d6c662f6c799ad","bytes":11486,"sha256":"afc15a734ac87a970904a8a4e9222c7b5d77375e9bbcf9e58f000d0fbfc86bf6"}),
	Object.freeze({"path":"internal/observe/schedule_test.go","mode":"100644","type":"blob","blob":"16aba963bd16e23c9827f98a96065d751a83de91","bytes":4483,"sha256":"d51b42e6d97e6d9ffb16cac36d59f2bd48a3dfe7b318fcffe2c4924584197bb4"}),
	Object.freeze({"path":"internal/reduce/draft.go","mode":"100644","type":"blob","blob":"538a179a82c1dfc5e35d020a771e082a981310e6","bytes":15117,"sha256":"d0fe169434826447a989e402818315cae5bb001547463bdbf1626b5de074e896"}),
	Object.freeze({"path":"internal/reduce/engine.go","mode":"100644","type":"blob","blob":"466d7df995dad740f48169e24339e141bc75b103","bytes":45232,"sha256":"71f98b6b0c61530f2a34b7eb037e8ca45bfa29a46c52eb52b5f0e46dba1d43eb"}),
	Object.freeze({"path":"internal/reduce/engine_test.go","mode":"100644","type":"blob","blob":"170497426540f2805035bb88ca9d496367de03f3","bytes":35582,"sha256":"7cff737e035633c323dccc1243786c0a033d365e903d1e15d1eef4723bb73789"}),
	Object.freeze({"path":"internal/reduce/identity.go","mode":"100644","type":"blob","blob":"2a70300cfb4627dfdf32a4125359413776644949","bytes":9029,"sha256":"1d7f596c83f3cc52cb9d04f02e30fff5077d6e8bf11b8b9029a4c7f9e80d1daa"}),
	Object.freeze({"path":"internal/reduce/model.go","mode":"100644","type":"blob","blob":"aa1bedd075485027fda3822b1b4cca379aac7671","bytes":28081,"sha256":"326c814e427b4bd5f04bb24249f904f265b4884a52317b7af5a2920b670e4583"}),
	Object.freeze({"path":"internal/reduce/model_test.go","mode":"100644","type":"blob","blob":"7bfe322d2d3d6b5dda973aa535e8506eccfaa735","bytes":24139,"sha256":"5d097bac90c04fff0cc368f53035221eb8f678d661dfab10d591ca4865231cce"}),
	Object.freeze({"path":"internal/reduce/wire.go","mode":"100644","type":"blob","blob":"fccde68bd956d1b22e602f664aced5967fdbf8d8","bytes":32455,"sha256":"869a8bc8e9637c775d388bd81ed5e35625f3b7b8cf0ce79156acc49d75dfdc04"}),
	Object.freeze({"path":"internal/reduce/wire_test.go","mode":"100644","type":"blob","blob":"be4a1efc1ab2da36e9b18089e01c2f59b0badd6e","bytes":32497,"sha256":"b065b81f653e21eb8a704dbf3f0741f4df174d3014cd72e9b787984450763566"}),
	Object.freeze({"path":"internal/reduction/finalize.go","mode":"100644","type":"blob","blob":"b760a3b2601a852d83d532a00cabdfa3b6e0873a","bytes":12545,"sha256":"f6bd1340129aec3b18c931102e009757a5c9d453e2d5bf66c4b88d9dd6ca8b8f"}),
	Object.freeze({"path":"internal/reduction/finalize_test.go","mode":"100644","type":"blob","blob":"8d57ab31c902e2f9bdf25aebcd6de491c4fecee1","bytes":20937,"sha256":"f6ed0acdf0c303fd6a4a3716d72f991a626d6b559d7d886c0ad036e9af8fd802"}),
	Object.freeze({"path":"tools/check-u5-architecture-selftest.mjs","mode":"100644","type":"blob","blob":"7d8bccd36c658070b2c7ab4b34e25b0f01c405d7","bytes":12990,"sha256":"1353ce0647619e4d883a5db4e4466dcec375358b8b89d42adf9772950bec599a"}),
	Object.freeze({"path":"tools/check-u5-architecture.mjs","mode":"100644","type":"blob","blob":"3712a8ab81526d99b49567959282eb38d0b10027","bytes":26178,"sha256":"b3742b510ca3074b841ed9be501216423d128ba871e6697555dad29646d55c8f"}),
	Object.freeze({"path":"tools/check-u6-architecture-selftest.mjs","mode":"100644","type":"blob","blob":"03945f91d3fcaa62728e70c0f5e72ff7df777bd4","bytes":74297,"sha256":"86ce6c57dcffea2610c2e135e869ec2bff725270d49ec642cb473f878a93ba13"}),
	Object.freeze({"path":"tools/check-u6-architecture.mjs","mode":"100644","type":"blob","blob":"fa0fc0c7c629fdbc7347697850207473c38207c6","bytes":132760,"sha256":"4eae6f6be02f3666b46a7712461334ca5c82d2c4b5058950d8f94c6ca721d5e1"}),
]);
const protectedAuthority = Object.freeze({
	commit: "4cef12b38cfcd857593a21db5952f2dfb2dfc274",
	tree: "b50aee79f41c3f498524d89d425d6dd8f4009702",
	fileCount: 67,
	totalBytes: 1298581,
	gitProjectionSha256: "583c9856f4516842fed9e451719f9155814fc6b95c286bedcac38f42f5fd7ee0",
	byteProjectionSha256: "93b57b611743ad059fae0d7e449197b96ee71d9088b835025c7218b5d683189d",
	rawProjectionSha256: "2e92efbc88e3e335caa78a8f520a385258d40f477cebcee801bd03e93e335a25",
});

const forbiddenAuthorityNames = Object.freeze([
	"CandidateExecutionKey",
	"CandidateOutcomeMap",
	"Digest",
	"ProjectionFingerprint",
	"Eligibility",
	"StabilityClassification",
	"ReductionGrade",
	"Choicepoint",
	"Ruling",
	"DecisionRecord",
	"ContractExecution",
	"WorldPlan",
	"WorldInstance",
]);
const forbiddenAuthorityConstants = Object.freeze([
	"OBSERVED_STABLE",
	"PRESERVES",
	"CHANGES",
	"UNRESOLVED",
	"ONE_MINIMAL_UNDER",
	"BEST_KNOWN",
	"CONFORMS",
	"CONTRADICTS",
]);

const requiredMainImportsByPhase = Object.freeze({
	U7P: Object.freeze([]),
	U7A: Object.freeze([`${modulePrefix}internal/reference/app`]),
	U7B: Object.freeze([`${modulePrefix}internal/reference/app`, `${modulePrefix}internal/reference/httpstudy`]),
	U7C: Object.freeze([`${modulePrefix}internal/reference/app`, `${modulePrefix}internal/reference/httpstudy`, `${modulePrefix}internal/reference/clistudy`]),
	U7D: Object.freeze([`${modulePrefix}internal/reference/app`, `${modulePrefix}internal/reference/httpstudy`, `${modulePrefix}internal/reference/clistudy`, `${modulePrefix}internal/reference/reproduce`]),
});

const javascriptASTProgram = String.raw`
const acorn = require("internal/deps/acorn/acorn/dist/acorn");
const fs = require("node:fs");
const entries = JSON.parse(fs.readFileSync(0, "utf8"));

function walk(node, visit, parent = null, parentKey = null) {
  if (!node || typeof node !== "object") return;
  if (typeof node.type === "string") visit(node, parent, parentKey);
  for (const [key, value] of Object.entries(node)) {
    if (key === "start" || key === "end" || key === "loc") continue;
    if (Array.isArray(value)) for (const child of value) walk(child, visit, node, key);
    else if (value && typeof value === "object") walk(value, visit, node, key);
  }
}

function propertyName(node) {
  if (!node || node.type !== "MemberExpression") return null;
  if (!node.computed && node.property.type === "Identifier") return node.property.name;
  if (node.computed && node.property.type === "Literal" && typeof node.property.value === "string") return node.property.value;
  return null;
}

function memberPath(node) {
  if (!node) return null;
  if (node.type === "Identifier") return node.name;
  if (node.type === "MetaProperty") return node.meta.name + "." + node.property.name;
  if (node.type !== "MemberExpression") return null;
  const object = memberPath(node.object);
  const property = propertyName(node);
  return object && property ? object + "." + property : null;
}

const rows = [];
for (const entry of entries) {
  const ast = acorn.parse(entry.source, { ecmaVersion: "latest", sourceType: "module", allowHashBang: true });
  const staticImports = [];
  const reexports = [];
  const dynamicImports = [];
  const requireCalls = [];
  const forbiddenResolvers = [];
  const fetchCalls = [];
  for (const statement of ast.body) {
    if (statement.type === "ImportDeclaration") staticImports.push(statement.source.value);
    if ((statement.type === "ExportNamedDeclaration" || statement.type === "ExportAllDeclaration") && statement.source) {
      reexports.push(statement.source.value);
    }
  }
  walk(ast, (node, parent, parentKey) => {
    if (node.type === "ImportExpression") {
      dynamicImports.push(node.source?.type === "Literal" && typeof node.source.value === "string" ? node.source.value : "<dynamic>");
    }
    if (node.type === "MemberExpression") {
      const path = memberPath(node);
      const object = memberPath(node.object);
      if (["module.createRequire", "process.getBuiltinModule", "process.binding", "import.meta.resolve", "WebAssembly.compile", "WebAssembly.instantiate", "globalThis.process", "globalThis.WebAssembly", "globalThis.eval", "globalThis.Function", "global.process", "global.WebAssembly", "global.eval", "global.Function"].includes(path)) {
        forbiddenResolvers.push(path);
      }
      if (node.computed && propertyName(node) === null && ["process", "globalThis", "global", "import.meta", "WebAssembly"].includes(object)) {
        forbiddenResolvers.push(object + "[<dynamic>]");
      }
      if (["globalThis.fetch", "globalThis.WebSocket", "global.fetch", "global.WebSocket"].includes(path)) fetchCalls.push(path);
    }
    if (node.type === "MetaProperty" && node.meta?.name === "import" && node.property?.name === "meta" &&
        !(parent?.type === "MemberExpression" && parentKey === "object" && !parent.computed)) {
      forbiddenResolvers.push("import.meta<alias>");
    }
    if (node.type === "Identifier" && node.name === "process" &&
        !(parent?.type === "MemberExpression" && parentKey === "object")) {
      forbiddenResolvers.push("process<alias>");
    }
    if (node.type === "Identifier" && ["globalThis", "global", "WebAssembly"].includes(node.name) &&
        !(parent?.type === "MemberExpression" && parentKey === "object") &&
        !(parent?.type === "Property" && parentKey === "key" && !parent.computed)) {
      forbiddenResolvers.push(node.name + "<alias>");
    }
    if (node.type === "Identifier" && ["fetch", "WebSocket"].includes(node.name) &&
        !(parent?.type === "MemberExpression" && parentKey === "property" && !parent.computed) &&
        !(parent?.type === "Property" && parentKey === "key" && !parent.computed)) {
      fetchCalls.push(node.name);
    }
    if (node.type === "Identifier" && ["eval", "Function"].includes(node.name) &&
        !(parent?.type === "MemberExpression" && parentKey === "property" && !parent.computed) &&
        !(parent?.type === "Property" && parentKey === "key" && !parent.computed)) {
      forbiddenResolvers.push(node.name);
    }
    if (node.type === "CallExpression" || node.type === "NewExpression") {
      const callee = memberPath(node.callee) || (node.callee?.type === "Identifier" ? node.callee.name : "<dynamic>");
      if (callee === "require") requireCalls.push(node.arguments?.[0]?.type === "Literal" ? node.arguments[0].value : "<dynamic>");
      if (["eval", "Function", "module.createRequire", "process.getBuiltinModule", "process.binding", "import.meta.resolve", "WebAssembly.compile", "WebAssembly.instantiate"].includes(callee)) forbiddenResolvers.push(callee);
      if (callee === "fetch" || callee === "globalThis.fetch" || callee === "WebSocket" || callee === "globalThis.WebSocket") fetchCalls.push(callee);
    }
  });
  rows.push({ path: entry.path, staticImports, reexports, dynamicImports, requireCalls, forbiddenResolvers, fetchCalls });
}
process.stdout.write(JSON.stringify(rows));
`;

class ArchitectureError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

class UsageError extends Error {
	constructor(detail) {
		super(`U7_ARCH_USAGE: ${detail}`);
	}
}

function byteCompare(left, right) {
	return Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8"));
}

function slashPath(value) {
	return value.split(sep).join("/");
}

function sameStat(left, right) {
	return left.dev === right.dev && left.ino === right.ino && left.size === right.size &&
		left.mode === right.mode && left.mtimeMs === right.mtimeMs && left.nlink === right.nlink;
}

function modeToken(stat) {
	return (stat.mode & 0o111) === 0 ? "100644" : "100755";
}

async function lstatOptional(path) {
	try {
		return await lstat(path);
	} catch (error) {
		if (error?.code === "ENOENT") return undefined;
		throw new ArchitectureError("U7_ARCH_STAT_FAILED", `${slashPath(relative(repositoryRoot, path))}: ${error?.code ?? error}`);
	}
}

async function readRegularNoFollow(root, path, expectedMode = undefined) {
	const absolute = resolve(root, path);
	const fromRoot = relative(root, absolute);
	if (fromRoot === ".." || fromRoot.startsWith(`..${sep}`) || resolve(root, fromRoot) !== absolute) {
		throw new ArchitectureError("U7_ARCH_PATH_ESCAPE", path);
	}
	const before = await lstatOptional(absolute);
	if (before === undefined) throw new ArchitectureError("U7_ARCH_MISSING", path);
	if (before.isSymbolicLink() || !before.isFile()) throw new ArchitectureError("U7_ARCH_NONREGULAR", path);
	if (before.nlink !== 1) throw new ArchitectureError("U7_ARCH_LINK_COUNT", `${path}: ${before.nlink}`);
	if (expectedMode !== undefined && modeToken(before) !== expectedMode) {
		throw new ArchitectureError("U7_ARCH_MODE", `${path}: ${modeToken(before)} != ${expectedMode}`);
	}
	let handle;
	try {
		handle = await open(absolute, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const opened = await handle.stat();
		if (!opened.isFile() || !sameStat(before, opened)) throw new ArchitectureError("U7_ARCH_FILE_CHANGED", path);
		const bytes = await handle.readFile();
		const after = await handle.stat();
		if (!sameStat(opened, after)) throw new ArchitectureError("U7_ARCH_FILE_CHANGED", path);
		return Object.freeze({ path, mode: modeToken(opened), bytes, stat: opened });
	} catch (error) {
		if (error instanceof ArchitectureError) throw error;
		throw new ArchitectureError("U7_ARCH_READ_FAILED", `${path}: ${error?.code ?? error}`);
	} finally {
		await handle?.close();
	}
}

function decodeUTF8(bytes, path) {
	try {
		return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch {
		throw new ArchitectureError("U7_ARCH_INVALID_UTF8", path);
	}
}

function sha256(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

function gitBlobOID(bytes) {
	return createHash("sha1").update(`blob ${bytes.length}\0`).update(bytes).digest("hex");
}

function manifestDigest(records) {
	const digest = createHash("sha256");
	for (const record of records) {
		digest.update(record.path);
		digest.update("\0");
		digest.update(record.mode);
		digest.update("\0");
		digest.update(String(record.bytes.length));
		digest.update("\0");
		digest.update(sha256(record.bytes));
		digest.update("\n");
	}
	return digest.digest("hex");
}

async function enumerateDirectory(root, relativeRoot) {
	const absoluteRoot = resolve(root, relativeRoot);
	const rootStat = await lstatOptional(absoluteRoot);
	if (rootStat === undefined) return undefined;
	if (rootStat.isSymbolicLink() || !rootStat.isDirectory()) throw new ArchitectureError("U7_ARCH_ROOT_NONDIRECTORY", relativeRoot);
	const paths = [];
	async function walk(absolute) {
		let entries;
		try {
			entries = await readdir(absolute, { withFileTypes: true });
		} catch (error) {
			throw new ArchitectureError("U7_ARCH_READDIR_FAILED", `${slashPath(relative(root, absolute))}: ${error?.code ?? error}`);
		}
		entries.sort((left, right) => byteCompare(left.name, right.name));
		for (const entry of entries) {
			const child = join(absolute, entry.name);
			const path = slashPath(relative(root, child));
			const childStat = await lstatOptional(child);
			if (childStat === undefined) throw new ArchitectureError("U7_ARCH_FILE_CHANGED", path);
			if (childStat.isSymbolicLink()) throw new ArchitectureError("U7_ARCH_NONREGULAR", path);
			if (childStat.isDirectory()) await walk(child);
			else if (childStat.isFile()) paths.push(path);
			else throw new ArchitectureError("U7_ARCH_NONREGULAR", path);
		}
	}
	await walk(absoluteRoot);
	return paths;
}

function validateProtectedRows() {
	if (protectedRows.length !== protectedAuthority.fileCount) {
		throw new ArchitectureError("U7_ARCH_PROTECTED_TABLE", `count ${protectedRows.length}`);
	}
	let totalBytes = 0;
	const gitProjection = createHash("sha256");
	const byteProjection = createHash("sha256");
	const rawProjection = createHash("sha256");
	for (const [index, row] of protectedRows.entries()) {
		if (!isDeepStrictEqual(Object.keys(row), ["path", "mode", "type", "blob", "bytes", "sha256"]) ||
			typeof row.path !== "string" || row.mode !== "100644" || row.type !== "blob" ||
			!/^[0-9a-f]{40}$/u.test(row.blob) || !Number.isSafeInteger(row.bytes) || row.bytes < 1 || !/^[0-9a-f]{64}$/u.test(row.sha256)) {
			throw new ArchitectureError("U7_ARCH_PROTECTED_TABLE", `row ${index}`);
		}
		if (index > 0 && byteCompare(protectedRows[index - 1].path, row.path) >= 0) {
			throw new ArchitectureError("U7_ARCH_PROTECTED_TABLE", `order ${row.path}`);
		}
		if (![...protectedRoots, ...protectedToolPaths].some((prefix) => row.path === prefix || row.path.startsWith(`${prefix}/`))) {
			throw new ArchitectureError("U7_ARCH_PROTECTED_TABLE", `jurisdiction ${row.path}`);
		}
		totalBytes += row.bytes;
		gitProjection.update(`${row.path}\0${row.mode}\0${row.type}\0${row.blob}\0`);
		byteProjection.update(`${row.path}\0${row.mode}\0${row.bytes}\0`);
		rawProjection.update(`${row.path}\0${row.mode}\0${row.type}\0${row.blob}\0${row.sha256}\0`);
	}
	if (totalBytes !== protectedAuthority.totalBytes || gitProjection.digest("hex") !== protectedAuthority.gitProjectionSha256 ||
		byteProjection.digest("hex") !== protectedAuthority.byteProjectionSha256 || rawProjection.digest("hex") !== protectedAuthority.rawProjectionSha256) {
		throw new ArchitectureError("U7_ARCH_PROTECTED_TABLE", "aggregate authority mismatch");
	}
}

async function collectProtectedManifest(root = repositoryRoot) {
	validateProtectedRows();
	const observedPaths = [];
	for (const protectedRoot of protectedRoots) {
		const rows = await enumerateDirectory(root, protectedRoot);
		if (rows === undefined) throw new ArchitectureError("U7_ARCH_PROTECTED_MISSING", protectedRoot);
		observedPaths.push(...rows);
	}
	for (const path of protectedToolPaths) {
		if (await lstatOptional(resolve(root, path)) === undefined) throw new ArchitectureError("U7_ARCH_PROTECTED_MISSING", path);
		observedPaths.push(path);
	}
	observedPaths.sort(byteCompare);
	const expectedPaths = protectedRows.map((row) => row.path);
	if (!isDeepStrictEqual(observedPaths, expectedPaths)) {
		const missing = expectedPaths.find((path) => !observedPaths.includes(path));
		const extra = observedPaths.find((path) => !expectedPaths.includes(path));
		throw new ArchitectureError(missing ? "U7_ARCH_PROTECTED_MISSING" : "U7_ARCH_PROTECTED_EXTRA", missing ?? extra ?? "inventory");
	}
	const records = [];
	for (const row of protectedRows) {
		const record = await readRegularNoFollow(root, row.path, row.mode);
		if (record.bytes.length !== row.bytes || sha256(record.bytes) !== row.sha256 || gitBlobOID(record.bytes) !== row.blob) {
			throw new ArchitectureError("U7_ARCH_PROTECTED_BYTES", row.path);
		}
		records.push(record);
	}
	return Object.freeze({ digest: manifestDigest(records), records: Object.freeze(records) });
}

function cumulativeSurface(phase) {
	const index = phases.indexOf(phase);
	if (index < 0) throw new UsageError(`phase must be one of ${phases.join(",")}`);
	const seen = new Set();
	const result = [];
	for (const current of phases.slice(0, index + 1)) {
		for (const path of declaredSurfaceByPhase[current]) {
			if (!seen.has(path)) {
				seen.add(path);
				result.push(path);
			}
		}
	}
	return Object.freeze(result);
}

function isGovernedPath(path) {
	return governedRoots.some((root) => path.startsWith(`${root}/`)) || governedDriverPattern.test(path);
}

function validateSpecificationSurface(value) {
	if (!value || typeof value !== "object" || value.schema_version !== "countershape/u7-unit-paths/v1" || !Array.isArray(value.units)) {
		throw new ArchitectureError("U7_ARCH_SPECIFICATION", "schema");
	}
	if (!isDeepStrictEqual(value.units.map((unit) => unit.id), ["U7P", "U7A", "U7B", "U7C", "U7D", "U7R"])) {
		throw new ArchitectureError("U7_ARCH_SPECIFICATION", "unit order");
	}
	for (const phase of phases) {
		const unit = value.units.find((candidate) => candidate.id === phase);
		const identity = phaseIdentity[phase];
		if (!unit || unit.parent !== identity.parent || unit.verification_profile !== identity.profile ||
			unit.product_authority !== identity.productAuthority || unit.product_behavior !== identity.productBehavior ||
			unit.u7d_receipt_state !== identity.receipt || !isDeepStrictEqual(unit.allowed_paths?.filter(isGovernedPath), declaredSurfaceByPhase[phase]) ||
			!isDeepStrictEqual(unit.required_paths, unit.allowed_paths)) {
			throw new ArchitectureError("U7_ARCH_SPECIFICATION", phase);
		}
	}
	const u7p = value.units[0];
	const u7d = value.units[4];
	for (const path of ["tools/check-u7-architecture.mjs", "tools/check-u7-architecture-selftest.mjs"]) {
		if (!u7p.allowed_paths.includes(path) || value.units.slice(1).some((unit) => unit.allowed_paths.includes(path))) {
			throw new ArchitectureError("U7_ARCH_SPECIFICATION", `oracle ownership ${path}`);
		}
	}
	const exactArchitectureClaims = [
		["U7P", "U7P dormant future-surface architecture conformance", ["/opt/homebrew/bin/node", "tools/check-u7-architecture.mjs", "--phase", "U7P"]],
		["U7P", "U7P architecture authority defensive self-test", ["/opt/homebrew/bin/node", "tools/check-u7-architecture-selftest.mjs", "--phase", "U7P"]],
		["U7D", "U7D two-domain architecture boundary", ["/opt/homebrew/bin/node", "tools/check-u7-architecture.mjs", "--phase", "U7D"]],
		["U7D", "U7D two-domain architecture defensive self-test", ["/opt/homebrew/bin/node", "tools/check-u7-architecture-selftest.mjs", "--phase", "U7D"]],
	];
	for (const [phase, label, command] of exactArchitectureClaims) {
		const unit = phase === "U7P" ? u7p : u7d;
		if (!unit.claims.some((claim) => claim.type === "tests-pass" && claim.label === label && isDeepStrictEqual(claim.command, command))) {
			throw new ArchitectureError("U7_ARCH_SPECIFICATION", `claim ${label}`);
		}
	}
}

async function loadSpecification() {
	const record = await readRegularNoFollow(repositoryRoot, "spec/verification/u7-unit-paths.json", "100644");
	let value;
	try {
		value = JSON.parse(decodeUTF8(record.bytes, "spec/verification/u7-unit-paths.json"));
	} catch (error) {
		if (error instanceof ArchitectureError) throw error;
		throw new ArchitectureError("U7_ARCH_SPECIFICATION", "invalid JSON");
	}
	validateSpecificationSurface(value);
	return Object.freeze({ bytes: record.bytes, sha256: sha256(record.bytes), value });
}

async function collectGovernedManifest(root, phase) {
	const expectedPaths = cumulativeSurface(phase);
	const expected = new Set(expectedPaths);
	const observedPaths = [];
	for (const governedRoot of governedRoots) {
		const rows = await enumerateDirectory(root, governedRoot);
		const expectedUnderRoot = expectedPaths.some((path) => path.startsWith(`${governedRoot}/`));
		if (rows === undefined) {
			if (expectedUnderRoot) throw new ArchitectureError("U7_ARCH_MISSING", governedRoot);
			continue;
		}
		if (!expectedUnderRoot) throw new ArchitectureError("U7_ARCH_FUTURE_SURFACE", governedRoot);
		observedPaths.push(...rows);
	}
	const toolsRoot = resolve(root, "tools");
	const toolsStat = await lstatOptional(toolsRoot);
	if (toolsStat === undefined || toolsStat.isSymbolicLink() || !toolsStat.isDirectory()) throw new ArchitectureError("U7_ARCH_ROOT_NONDIRECTORY", "tools");
	const toolEntries = await readdir(toolsRoot, { withFileTypes: true });
	for (const entry of toolEntries) {
		const path = `tools/${entry.name}`;
		if (!governedDriverPattern.test(path)) continue;
		const stat = await lstatOptional(resolve(root, path));
		if (stat === undefined || stat.isSymbolicLink() || !stat.isFile()) throw new ArchitectureError("U7_ARCH_NONREGULAR", path);
		observedPaths.push(path);
	}
	observedPaths.sort(byteCompare);
	const expectedSorted = [...expectedPaths].sort(byteCompare);
	if (!isDeepStrictEqual(observedPaths, expectedSorted)) {
		const missing = expectedSorted.find((path) => !observedPaths.includes(path));
		const extra = observedPaths.find((path) => !expected.has(path));
		throw new ArchitectureError(missing ? "U7_ARCH_MISSING" : "U7_ARCH_EXTRA", missing ?? extra ?? "inventory");
	}
	const records = [];
	for (const path of expectedSorted) {
		const expectedMode = "100644";
		const record = await readRegularNoFollow(root, path, expectedMode);
		if (!path.endsWith(".go") && !path.endsWith(".mjs")) throw new ArchitectureError("U7_ARCH_FILE_TYPE", path);
		decodeUTF8(record.bytes, path);
		records.push(record);
	}
	return Object.freeze({ digest: manifestDigest(records), records: Object.freeze(records) });
}

function lexicalViews(source, path) {
	const code = source.split("");
	const commentless = source.split("");
	let state = "code";
	for (let index = 0; index < source.length; index += 1) {
		const character = source[index];
		const next = source[index + 1] ?? "";
		if (state === "code") {
			if (character === "/" && next === "/") {
				code[index] = code[index + 1] = commentless[index] = commentless[index + 1] = " ";
				index += 1;
				state = "line-comment";
			} else if (character === "/" && next === "*") {
				code[index] = code[index + 1] = commentless[index] = commentless[index + 1] = " ";
				index += 1;
				state = "block-comment";
			} else if (character === '"' || character === "'" || character === "`") {
				code[index] = " ";
				state = character === '"' ? "string" : character === "'" ? "rune" : "raw";
			}
		} else if (state === "line-comment") {
			code[index] = commentless[index] = character === "\n" ? "\n" : " ";
			if (character === "\n") state = "code";
		} else if (state === "block-comment") {
			code[index] = commentless[index] = character === "\n" ? "\n" : " ";
			if (character === "*" && next === "/") {
				code[index + 1] = commentless[index + 1] = " ";
				index += 1;
				state = "code";
			}
		} else if (state === "raw") {
			code[index] = character === "\n" ? "\n" : " ";
			if (character === "`") state = "code";
		} else {
			code[index] = character === "\n" ? "\n" : " ";
			if (character === "\\") {
				index += 1;
				if (index < code.length) code[index] = source[index] === "\n" ? "\n" : " ";
			} else if ((state === "string" && character === '"') || (state === "rune" && character === "'")) {
				state = "code";
			}
		}
	}
	if (state === "line-comment") state = "code";
	if (state !== "code") throw new ArchitectureError("U7_ARCH_GO_LEXICAL", `${path}: unterminated ${state}`);
	return Object.freeze({ code: code.join(""), commentless: commentless.join("") });
}

function goTokens(source, path) {
	const tokens = [];
	for (let index = 0; index < source.length;) {
		const character = source[index];
		if (/\s/u.test(character)) {
			let sawNewline = false;
			while (index < source.length && /\s/u.test(source[index])) {
				if (source[index] === "\n") sawNewline = true;
				index += 1;
			}
			const previous = tokens.at(-1);
			if (sawNewline && previous !== undefined && previous.value !== ";" &&
				(["ident", "string", "number"].includes(previous.type) || [")", "]", "}"].includes(previous.value))) {
				tokens.push({ type: "punct", value: ";" });
			}
			continue;
		}
		if (/[A-Za-z_]/u.test(character)) {
			let end = index + 1;
			while (end < source.length && /[A-Za-z0-9_]/u.test(source[end])) end += 1;
			tokens.push({ type: "ident", value: source.slice(index, end) });
			index = end;
			continue;
		}
		if (/[0-9]/u.test(character)) {
			let end = index + 1;
			while (end < source.length && /[0-9A-Fa-f_xXobOB.]/u.test(source[end])) end += 1;
			tokens.push({ type: "number", value: source.slice(index, end) });
			index = end;
			continue;
		}
		if (character === '"') {
			let end = index + 1;
			for (; end < source.length; end += 1) {
				if (source[end] === "\\") end += 1;
				else if (source[end] === '"') break;
			}
			if (end >= source.length) throw new ArchitectureError("U7_ARCH_GO_LEXICAL", `${path}: import string`);
			tokens.push({ type: "string", value: source.slice(index, end + 1) });
			index = end + 1;
			continue;
		}
		if (character === "`") {
			const end = source.indexOf("`", index + 1);
			if (end < 0) throw new ArchitectureError("U7_ARCH_GO_LEXICAL", `${path}: raw string`);
			tokens.push({ type: "string", value: source.slice(index, end + 1) });
			index = end + 1;
			continue;
		}
		if (character === "'") {
			let end = index + 1;
			for (; end < source.length; end += 1) {
				if (source[end] === "\\") end += 1;
				else if (source[end] === "'") break;
			}
			if (end >= source.length) throw new ArchitectureError("U7_ARCH_GO_LEXICAL", `${path}: rune literal`);
			tokens.push({ type: "string", value: source.slice(index, end + 1) });
			index = end + 1;
			continue;
		}
		tokens.push({ type: "punct", value: character });
		index += 1;
	}
	return tokens;
}

function decodeGoString(token, path) {
	if (token.startsWith("`")) return token.slice(1, -1).replaceAll("\r", "");
	try {
		return JSON.parse(token);
	} catch {
		throw new ArchitectureError("U7_ARCH_GO_IMPORT", `${path}: invalid import literal`);
	}
}

function importedPackageBindings(commentless, path) {
	const tokens = goTokens(commentless, path);
	const imports = [];
	for (let index = 0; index < tokens.length; index += 1) {
		if (tokens[index].type !== "ident" || tokens[index].value !== "import") continue;
		index += 1;
		if (tokens[index]?.value === "(") {
			index += 1;
			while (index < tokens.length && tokens[index].value !== ")") {
				if (tokens[index].value === ";") {
					index += 1;
					continue;
				}
				let alias;
				if (tokens[index].type === "ident" || tokens[index].value === ".") alias = tokens[index++].value;
				if (alias === ".") throw new ArchitectureError("U7_ARCH_DOT_IMPORT", path);
				if (tokens[index]?.type !== "string") throw new ArchitectureError("U7_ARCH_GO_IMPORT", `${path}: import block`);
				const imported = decodeGoString(tokens[index++].value, path);
				imports.push(Object.freeze({ alias: alias ?? imported.split("/").at(-1), path: imported }));
			}
			if (tokens[index]?.value !== ")") throw new ArchitectureError("U7_ARCH_GO_IMPORT", `${path}: unterminated import block`);
		} else {
			let alias;
			if (tokens[index]?.type === "ident" || tokens[index]?.value === ".") alias = tokens[index++].value;
			if (alias === ".") throw new ArchitectureError("U7_ARCH_DOT_IMPORT", path);
			if (tokens[index]?.type !== "string") throw new ArchitectureError("U7_ARCH_GO_IMPORT", `${path}: import`);
			const imported = decodeGoString(tokens[index].value, path);
			imports.push(Object.freeze({ alias: alias ?? imported.split("/").at(-1), path: imported }));
		}
	}
	return Object.freeze(imports);
}

function declarationNames(tokens, keyword) {
	const names = [];
	for (let index = 0; index < tokens.length; index += 1) {
		const token = tokens[index];
		if (token.type !== "ident" || token.value !== keyword) continue;
		index += 1;
		while (tokens[index]?.value === ";") index += 1;
		if (tokens[index]?.value !== "(") {
			while (tokens[index]?.type === "ident") {
				names.push(tokens[index].value);
				index += 1;
				if (tokens[index]?.value !== ",") break;
				index += 1;
			}
			continue;
		}
		let paren = 1;
		let nestedCurly = 0;
		let bracket = 0;
		let expectName = true;
		let nameListOpen = keyword === "const";
		for (index += 1; index < tokens.length && paren > 0; index += 1) {
			const current = tokens[index];
			if (paren === 1 && nestedCurly === 0 && bracket === 0 && expectName && current.type === "ident") {
				names.push(current.value);
				expectName = false;
				continue;
			}
			if (current.value === "(") paren += 1;
			else if (current.value === ")") paren -= 1;
			else if (current.value === "{") nestedCurly += 1;
			else if (current.value === "}") nestedCurly -= 1;
			else if (current.value === "[") bracket += 1;
			else if (current.value === "]") bracket -= 1;
			else if (current.value === ";" && paren === 1 && nestedCurly === 0 && bracket === 0) {
				expectName = true;
				nameListOpen = keyword === "const";
			} else if (current.value === "," && paren === 1 && nestedCurly === 0 && bracket === 0 && nameListOpen) {
				expectName = true;
			} else if (current.value === "=" && paren === 1 && nestedCurly === 0 && bracket === 0) {
				nameListOpen = false;
			}
		}
	}
	return Object.freeze(names);
}

function expectedPackage(path) {
	if (path.startsWith("cmd/countershape/")) return "main";
	if (path.startsWith("internal/reference/app/")) return "app";
	if (path.startsWith("internal/reference/httpstudy/")) return "httpstudy";
	if (path.startsWith("internal/reference/clistudy/")) return "clistudy";
	if (path.startsWith("internal/reference/reproduce/")) return "reproduce";
	if (path.startsWith("testkit/reference/")) return "reference";
	return undefined;
}

function sourceLayer(path) {
	if (path.startsWith("cmd/countershape/")) return "cmd";
	if (path.startsWith("internal/reference/app/")) return "app";
	if (path.startsWith("internal/reference/httpstudy/")) return "httpstudy";
	if (path.startsWith("internal/reference/clistudy/")) return "clistudy";
	if (path.startsWith("internal/reference/reproduce/")) return "reproduce";
	if (path.startsWith("testkit/reference/")) return "fixture";
	return "driver";
}

function localImport(importPath) {
	return importPath.startsWith(modulePrefix) ? importPath.slice(modulePrefix.length) : undefined;
}

function isThirdPartyImport(importPath) {
	if (importPath.startsWith(modulePrefix)) return false;
	const first = importPath.split("/")[0];
	return first.includes(".");
}

function requireExactLocalImports(path, imports, expected) {
	const actual = imports.filter((value) => value.startsWith(modulePrefix));
	for (const required of expected) {
		if (!actual.includes(required)) throw new ArchitectureError("U7_ARCH_REQUIRED_EDGE", `${path}: ${required}`);
	}
}

function isNeutralNetworkImport(imported) {
	return imported === "net" || imported.startsWith("net/") || imported === "crypto/tls" || imported.startsWith("crypto/tls/");
}

function validateImportEdge(path, imports, code, phase) {
	const layer = sourceLayer(path);
	const processOwner = path.endsWith("_test.go") || path === "internal/reference/app/run_darwin.go" || path === "internal/reference/reproduce/run_darwin.go";
	for (const imported of imports) {
		if (isThirdPartyImport(imported)) throw new ArchitectureError("U7_ARCH_THIRD_PARTY_IMPORT", `${path}: ${imported}`);
		if (["C", "unsafe", "plugin"].includes(imported)) throw new ArchitectureError("U7_ARCH_UNSAFE_IMPORT", `${path}: ${imported}`);
		const local = localImport(imported);
		if (layer === "cmd") {
			if (imported === "os/exec" || isNeutralNetworkImport(imported) || (local !== undefined && !requiredMainImportsByPhase[phase].includes(imported))) {
				throw new ArchitectureError("U7_ARCH_CMD_EDGE", `${path}: ${imported}`);
			}
		} else if (layer === "app") {
			if (isNeutralNetworkImport(imported) || (local !== undefined && (/^internal\/reference\/(httpstudy|clistudy|reproduce)(?:\/|$)/u.test(local) ||
				/^internal\/adapters\//u.test(local) || /^testkit\/reference(?:\/|$)/u.test(local) ||
				/^internal\/contract(exec|materialize)(?:\/|$)/u.test(local)))) {
				throw new ArchitectureError("U7_ARCH_APP_EDGE", `${path}: ${imported}`);
			}
		} else if (layer === "httpstudy") {
			if (local !== undefined && (/^internal\/reference\/(clistudy|reproduce)(?:\/|$)/u.test(local) || /^internal\/adapters\/cli(?:\/|$)/u.test(local))) {
				throw new ArchitectureError("U7_ARCH_HTTP_EDGE", `${path}: ${imported}`);
			}
		} else if (layer === "clistudy") {
			if (isNeutralNetworkImport(imported) || (local !== undefined && (/^internal\/reference\/(httpstudy|reproduce)(?:\/|$)/u.test(local) ||
				/^internal\/adapters\/http(?:\/|$)/u.test(local)))) {
				throw new ArchitectureError("U7_ARCH_CLI_EDGE", `${path}: ${imported}`);
			}
		} else if (layer === "reproduce") {
			if (isNeutralNetworkImport(imported) || (local !== undefined && !/^internal\/reference\/(app|httpstudy|clistudy)$/u.test(local))) {
				throw new ArchitectureError("U7_ARCH_REPRODUCE_EDGE", `${path}: ${imported}`);
			}
		} else if (layer === "fixture") {
			if (local !== undefined && /^internal\/reference(?:\/|$)/u.test(local)) throw new ArchitectureError("U7_ARCH_FIXTURE_EDGE", `${path}: ${imported}`);
			if (/^testkit\/reference\/http(?:_test)?\.go$/u.test(path) && local !== undefined && /^internal\/adapters\/cli(?:\/|$)/u.test(local)) {
				throw new ArchitectureError("U7_ARCH_FIXTURE_DOMAIN", `${path}: ${imported}`);
			}
			if (/^testkit\/reference\/cli(?:_test)?\.go$/u.test(path) && local !== undefined && /^internal\/adapters\/http(?:\/|$)/u.test(local)) {
				throw new ArchitectureError("U7_ARCH_FIXTURE_DOMAIN", `${path}: ${imported}`);
			}
			if (/^testkit\/reference\/cli(?:_test)?\.go$/u.test(path)) {
				if (isNeutralNetworkImport(imported)) throw new ArchitectureError("U7_ARCH_FIXTURE_EDGE", `${path}: ${imported}`);
			}
		}
		if (imported === "os/exec" && !path.endsWith("_test.go") && path !== "internal/reference/app/run_darwin.go" &&
			path !== "internal/reference/reproduce/run_darwin.go") {
			throw new ArchitectureError("U7_ARCH_PROCESS_OWNER", path);
		}
		if (imported === "syscall" && !processOwner) throw new ArchitectureError("U7_ARCH_PROCESS_OWNER", path);
	}
	if (layer === "httpstudy" && /\b(?:CLI|Cli)[A-Za-z0-9_]*\b/u.test(code)) throw new ArchitectureError("U7_ARCH_HTTP_CLI_COERCION", path);
	if (layer === "clistudy" && /\bHTTP[A-Za-z0-9_]*\b/u.test(code)) throw new ArchitectureError("U7_ARCH_CLI_HTTP_COERCION", path);
	if (path === "cmd/countershape/main.go") requireExactLocalImports(path, imports, requiredMainImportsByPhase[phase]);
	if (path === "internal/reference/httpstudy/study.go") requireExactLocalImports(path, imports, [
		`${modulePrefix}internal/reference/app`, `${modulePrefix}internal/adapters/http`, `${modulePrefix}testkit/reference`,
	]);
	if (path === "internal/reference/clistudy/study.go") requireExactLocalImports(path, imports, [
		`${modulePrefix}internal/reference/app`, `${modulePrefix}internal/adapters/cli`, `${modulePrefix}testkit/reference`,
	]);
	if (path === "internal/reference/reproduce/run_darwin.go") requireExactLocalImports(path, imports, [
		`${modulePrefix}internal/reference/app`, `${modulePrefix}internal/reference/httpstudy`, `${modulePrefix}internal/reference/clistudy`,
	]);
}

function validateGoSource(record, phase) {
	const source = decodeUTF8(record.bytes, record.path);
	if (/^\s*\/\/(?:go:build| \+build)\b/mu.test(source)) throw new ArchitectureError("U7_ARCH_BUILD_TAG", record.path);
	const views = lexicalViews(source, record.path);
	const tokens = goTokens(views.code, record.path);
	const packageNames = [];
	for (let index = 0; index < tokens.length - 1; index += 1) {
		if (tokens[index].type === "ident" && tokens[index].value === "package" && tokens[index + 1].type === "ident") {
			packageNames.push(tokens[index + 1].value);
		}
	}
	const expected = expectedPackage(record.path);
	if (packageNames.length !== 1 || packageNames[0] !== expected) {
		throw new ArchitectureError("U7_ARCH_PACKAGE", `${record.path}: ${packageNames.join(",") || "missing"} != ${expected}`);
	}
	const importBindings = importedPackageBindings(views.commentless, record.path);
	const imports = Object.freeze(importBindings.map((binding) => binding.path));
	validateImportEdge(record.path, imports, views.code, phase);
	const declaredTypes = declarationNames(tokens, "type");
	const declaredConstants = declarationNames(tokens, "const");
	const forbiddenType = declaredTypes.find((name) => forbiddenAuthorityNames.includes(name));
	const forbiddenConstant = declaredConstants.find((name) => forbiddenAuthorityConstants.includes(name));
	if (forbiddenType !== undefined) throw new ArchitectureError("U7_ARCH_AUTHORITY_REDECLARATION", `${record.path}: ${forbiddenType}`);
	if (forbiddenConstant !== undefined) throw new ArchitectureError("U7_ARCH_AUTHORITY_REDECLARATION", `${record.path}: ${forbiddenConstant}`);
	if (record.path === "internal/reference/app/render.go") {
		const authorityBindings = importBindings.filter((binding) => /^(?:github\.com\/nelsonwerd\/countershape\/internal\/(?:canon|domain|observe|compare|reduce|reduction|choice))(?:\/|$)/u.test(binding.path));
		if (authorityBindings.length > 0 || imports.some((value) => /^(?:crypto(?:\/|$)|hash(?:\/|$))/u.test(value)) ||
			/\b(?:AssessComparison|Canonicalize|CanonicalizeTyped|Classify|Compare|DigestBytes|DigestTyped|DigestValue|Fingerprint|MarshalTyped|MustDigest|NewAttempt|NewCandidateExecutionBinding|NewCandidateExecutionKey|NewComparisonEnvelope|NewDidrunReceipt|NewDigest|NewInstanceMeasurements|NewProjectionDefinitionBinding|NewProjectionFingerprint|NewWorldInstance|NewWorldPlan|ParseDigest|ParseProjectionFingerprint|ReceiptFromWire|Reduce|Sum256|Unreceipted|ValidateWorldPlanDeclaration)\s*\(/u.test(views.code)) {
			throw new ArchitectureError("U7_ARCH_RENDER_AUTHORITY", record.path);
		}
	}
	const processOwner = record.path.endsWith("_test.go") || record.path === "internal/reference/app/run_darwin.go" ||
		record.path === "internal/reference/reproduce/run_darwin.go";
	if (!processOwner && /\b(?:StartProcess|ForkExec|RawSyscall|Syscall)\b/u.test(views.code)) {
		throw new ArchitectureError("U7_ARCH_PROCESS_OWNER", record.path);
	}
	if (/^testkit\/reference\/http(?:_test)?\.go$/u.test(record.path) && /\b(?:CLI|Cli)[A-Za-z0-9_]*\b/u.test(views.code)) {
		throw new ArchitectureError("U7_ARCH_FIXTURE_DOMAIN", record.path);
	}
	if (/^testkit\/reference\/cli(?:_test)?\.go$/u.test(record.path) && /\bHTTP[A-Za-z0-9_]*\b/u.test(views.code)) {
		throw new ArchitectureError("U7_ARCH_FIXTURE_DOMAIN", record.path);
	}
	for (const match of source.matchAll(/(["`])https?:\/\/([^"`\s]+)\1/gu)) {
		const url = `http://${match[2]}`;
		const layer = sourceLayer(record.path);
		if (!(["httpstudy", "fixture"].includes(layer) && (url.startsWith("http://127.0.0.1") || url.startsWith("http://[::1]")) && match[0].startsWith('"http://'))) {
			throw new ArchitectureError("U7_ARCH_EXTERNAL_URL", record.path);
		}
	}
	return Object.freeze({ path: record.path, layer: sourceLayer(record.path), imports });
}

function validateNodeSources(records) {
	if (records.length === 0) return Object.freeze([]);
	const entries = records.map((record) => {
		const source = decodeUTF8(record.bytes, record.path);
		if (!source.startsWith("#!/usr/bin/env node\n")) throw new ArchitectureError("U7_ARCH_NODE_SHEBANG", record.path);
		return { path: record.path, source };
	});
	const result = spawnSync(process.execPath, ["--expose-internals", "-e", javascriptASTProgram], {
		cwd: repositoryRoot,
		input: JSON.stringify(entries),
		encoding: "utf8",
		timeout: 15_000,
		maxBuffer: 4 * 1024 * 1024,
		env: {
			PATH: process.env.PATH ?? "/usr/bin:/bin:/opt/homebrew/bin",
			HOME: process.env.HOME ?? repositoryRoot,
			TMPDIR: process.env.TMPDIR ?? "/tmp",
			LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1", NODE_OPTIONS: "", NODE_PATH: "",
		},
	});
	if (result.error || result.signal || result.status !== 0) {
		throw new ArchitectureError("U7_ARCH_NODE_AST", `${result.status ?? result.signal ?? result.error?.code ?? "failed"}`);
	}
	let rows;
	try {
		rows = JSON.parse(result.stdout);
	} catch {
		throw new ArchitectureError("U7_ARCH_NODE_AST", "malformed output");
	}
	if (!Array.isArray(rows) || rows.length !== entries.length) throw new ArchitectureError("U7_ARCH_NODE_AST", "row count");
	const allowedDriverBuiltins = new Set([
		"node:assert", "node:assert/strict", "node:buffer", "node:child_process", "node:crypto", "node:events",
		"node:fs", "node:fs/promises", "node:os", "node:path", "node:perf_hooks", "node:stream",
		"node:stream/promises", "node:string_decoder", "node:timers", "node:timers/promises", "node:url", "node:util",
	]);
	return Object.freeze(rows.map((row, index) => {
		const fields = ["staticImports", "reexports", "dynamicImports", "requireCalls", "forbiddenResolvers", "fetchCalls"];
		if (!row || row.path !== entries[index].path || fields.some((field) => !Array.isArray(row[field]) || row[field].some((value) => typeof value !== "string"))) {
			throw new ArchitectureError("U7_ARCH_NODE_AST", `row ${index}`);
		}
		if (row.reexports.length > 0 || row.dynamicImports.length > 0 || row.requireCalls.length > 0 || row.forbiddenResolvers.length > 0) {
			throw new ArchitectureError("U7_ARCH_NODE_DYNAMIC", row.path);
		}
		if (row.fetchCalls.length > 0) throw new ArchitectureError("U7_ARCH_NODE_NETWORK", `${row.path}: ${row.fetchCalls[0]}`);
		for (const imported of row.staticImports) {
			if (!imported.startsWith("node:")) {
				throw new ArchitectureError("U7_ARCH_NODE_DEPENDENCY", `${row.path}: ${imported}`);
			}
			if (!allowedDriverBuiltins.has(imported)) {
				if (/^node:(?:_?http|https|net|_?tls|dns|dgram|http2)(?:_|\/|$)/u.test(imported)) {
					throw new ArchitectureError("U7_ARCH_NODE_NETWORK", `${row.path}: ${imported}`);
				}
				throw new ArchitectureError("U7_ARCH_NODE_DYNAMIC", row.path);
			}
			if (/^node:(?:_?http|https|net|_?tls|dns|dgram|http2)(?:_|\/|$)/u.test(imported)) {
				throw new ArchitectureError("U7_ARCH_NODE_NETWORK", `${row.path}: ${imported}`);
			}
		}
		return Object.freeze({ path: row.path, layer: "driver", imports: Object.freeze([]) });
	}));
}

function validateReferenceCycles(sources) {
	const graph = new Map();
	for (const source of sources) {
		if (source.layer === "driver") continue;
		if (!graph.has(source.layer)) graph.set(source.layer, new Set());
		for (const imported of source.imports) {
			const local = localImport(imported);
			const match = /^(?:cmd\/countershape|internal\/reference\/(app|httpstudy|clistudy|reproduce)|testkit\/reference)$/u.exec(local ?? "");
			if (match) graph.get(source.layer).add(match[1] ?? (local === "cmd/countershape" ? "cmd" : "fixture"));
		}
	}
	const visiting = new Set();
	const visited = new Set();
	function visit(node) {
		if (visiting.has(node)) throw new ArchitectureError("U7_ARCH_IMPORT_CYCLE", node);
		if (visited.has(node)) return;
		visiting.add(node);
		for (const next of graph.get(node) ?? []) visit(next);
		visiting.delete(node);
		visited.add(node);
	}
	for (const node of [...graph.keys()].sort(byteCompare)) visit(node);
}

export async function checkArchitecture(phase) {
	if (!phases.includes(phase)) throw new UsageError(`phase must be one of ${phases.join(",")}`);
	const initialSpecification = await loadSpecification();
	const initialProtected = await collectProtectedManifest();
	const initialGoverned = await collectGovernedManifest(repositoryRoot, phase);
	const goSources = initialGoverned.records.filter((record) => record.path.endsWith(".go")).map((record) => validateGoSource(record, phase));
	const nodeSources = validateNodeSources(initialGoverned.records.filter((record) => record.path.endsWith(".mjs")));
	const sources = [...goSources, ...nodeSources];
	validateReferenceCycles(sources);
	const terminalGoverned = await collectGovernedManifest(repositoryRoot, phase);
	const terminalProtected = await collectProtectedManifest();
	const terminalSpecification = await loadSpecification();
	if (terminalGoverned.digest !== initialGoverned.digest || terminalProtected.digest !== initialProtected.digest ||
		terminalSpecification.sha256 !== initialSpecification.sha256) {
		throw new ArchitectureError("U7_ARCH_TERMINAL_DRIFT", phase);
	}
	return Object.freeze({ phase, governedCount: initialGoverned.records.length, protectedCount: initialProtected.records.length });
}

function parseArguments(argv) {
	if (!isDeepStrictEqual(argv.slice(0, 1), ["--phase"]) || argv.length !== 2 || !phases.includes(argv[1])) {
		throw new UsageError(`check-u7-architecture.mjs --phase ${phases.join("|")}`);
	}
	return argv[1];
}

async function main() {
	const phase = parseArguments(process.argv.slice(2));
	const result = await checkArchitecture(phase);
	process.stdout.write(`U7_ARCHITECTURE_OK phase=${result.phase} governed=${result.governedCount} protected=${result.protectedCount}\n`);
}

async function dispatch() {
	if (process.argv[1] === undefined) return;
	const requested = resolve(process.argv[1]);
	let actual;
	try {
		actual = await realpath(requested);
	} catch {
		return;
	}
	if (actual !== modulePath) return;
	if (requested !== modulePath) throw new UsageError("canonical checker path required");
	await main();
}

dispatch().catch((error) => {
	if (error instanceof UsageError) {
		process.stderr.write(`${error.message}\n`);
		process.exitCode = 2;
		return;
	}
	if (error instanceof ArchitectureError) {
		process.stderr.write(`${error.message}\n`);
		process.exitCode = 1;
		return;
	}
	process.stderr.write(`U7_ARCH_INTERNAL: ${error?.code ?? error?.message ?? String(error)}\n`);
	process.exitCode = 2;
});
