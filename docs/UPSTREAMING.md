<!-- This file is fork-only and is never offered upstream, so it deliberately
     does not add its vocabulary to .wordlist.txt — that file belongs to
     upstream and editing it here would create a permanent rebase conflict. -->
<!-- spellcheck-disable -->

# Upstreaming Checklist

This file is **fork-only**. It documents what `ROCm/gpu-operator` requires of a
contribution, so every branch here can be checked against it before it is
offered upstream.

It must never appear in an upstream pull request. See
[Branch hygiene](#branch-hygiene) for how that is guaranteed.

Researched against upstream `main` at `b007b235` (2026-08-14). Re-verify the
[Where the rules live](#where-the-rules-live) section if more than a release
has passed.

## Where the rules live

Upstream has no `CONTRIBUTING.md` of its own and no issue templates in the
repository. The governing documents are spread across three places:

| Document | Location | What it governs |
|---|---|---|
| ROCm contribution guide | `ROCm/ROCm` → `CONTRIBUTING.md` | The binding rules: PR process, tests, licensing grant |
| Pull request template | `ROCm/.github` → `.github/pull_request_template.md` | The required PR body sections |
| Issue templates | `ROCm/.github` → `.github/ISSUE_TEMPLATE/` | `issue_report`, `feature_request`, `documentation` |
| Developer guide | this repo → `docs/contributing/developer-guide.md` | Local build and toolchain setup |
| Documentation standards | this repo → `docs/contributing/documentation-standards.md` | Terminology and prose style |

### Licensing, sign-off, and the contributor agreement

- **There is no CLA and no DCO bot.** The ROCm contribution guide states the
  licence grant is implicit in opening the pull request: creating a PR is the
  act that licenses the contribution under the repository's `LICENSE`
  (Apache 2.0 here).
- **`Signed-off-by` is not enforced.** Only 59 of the last 200 upstream commits
  carry a sign-off trailer, and those are mostly the habit of contributors from
  other organizations. Adding one is harmless.
- **Decision for this fork: sign off anyway.** It costs nothing, it is a normal
  courtesy, and it removes any question about provenance if upstream policy
  changes later. Use `git commit -s`.

## The submission reality: GitHub is a mirror

This is the single most important thing to understand before offering work
upstream, because it changes what "merged" looks like.

Development happens in an internal AMD repository. The public GitHub repository
receives the results. Of the last 200 merged pull requests:

- 71 were opened by `ci-penbot-01`, a bot, titled `[CP <number>]` where the
  number is an internal pull request in the 1500-1650 range. `CP` is a
  cherry-pick.
- 71 were opened by the lead maintainer.
- The remainder are other AMD staff, ROCm documentation maintainers, and a
  small number of contributors from partner organizations.

The ROCm contribution guide confirms the mechanism: once a pull request is
approved, it "is brought onto internal CI systems and may be merged into the
component during our release cycle, as coordinated by the maintainer."

**What this means in practice:**

- Approval and merge are separate events, with a release cycle of lag between
  them.
- A contribution may land as a `[CP nnnn]` commit authored by the bot rather
  than as a merge of the submitted branch. The commit may not carry the
  original authorship.
- A pull request sitting open and approved is the normal state, not a stall.
  Do not close and resubmit.

### The stale warning, and the evidence against it

`docs/contributing/developer-guide.md` carries this warning:

> This project is not ready yet to accept the external developers commits.

Treat it as stale but live: it is a reason to open a discussion before
investing in a large change, not a reason to skip the work. The evidence that
the door is in fact open:

- **Pull request #465**, `feat: add HostNetwork field to DevicePlugin and
  MetricsExporter`, was contributed by an engineer with no AMD affiliation,
  reviewed and approved by the lead maintainer, and merged. It added optional
  fields to `DevicePluginSpec` and `MetricsExporterSpec` and wired them into
  both DaemonSets — structurally the same shape as the APU work in this fork.
- Contributors from Red Hat have merged work on the OpenShift and KMM paths.

**Cite #465 as precedent** in the first APU pull request. It is the closest
analogue and it establishes that CRD-surface additions from outside AMD are
accepted on their merits.

### Propose before building

The ROCm guide directs new features to
[GitHub Discussions](https://github.com/ROCm/ROCm/discussions) (Ideas category)
first. For APU support this is worth doing once, as a single discussion
covering the whole direction, before the first pull request — not once per
branch.

## Per-branch checklist

Work through this for every branch before it is offered. Nothing here is
optional unless marked.

### 1. Base and scope

- [ ] Branch is cut from `upstream/main`, not from this fork's `main`.
- [ ] Targets `main`. The generic ROCm guide says `develop`; this repository
      does not have one, and its default branch is `main`.
- [ ] Rebased onto current `upstream/main` with no merge commits.
- [ ] The branch does exactly one thing. If it can be described with an "and",
      split it.
- [ ] The branch is independently revertable — reverting it does not break any
      other branch offered before or after it.
- [ ] `docs/UPSTREAMING.md` is **not** in the diff. Neither is anything else
      fork-only.

### 2. Build and test gates

These are the checks that actually run on an upstream pull request.

- [ ] `make unit-test` passes. This is the `unit-test.yml` workflow. It depends
      on `vet`, so a vet failure fails the build.
- [ ] `make default` passes. This is the `ci.yml` workflow: a full containerized
      build of every target. It is slow; run it before offering, not on every
      commit.
- [ ] **New tests are actually inside the CI test scope.** See
      [The UNIT_TEST trap](#the-unit_test-trap) — this has already been
      confirmed to silently skip an existing test file.
- [ ] `make lint` passes (golangci-lint plus a `gofmt` gate). Not wired into any
      workflow and there is no `.golangci.yml` in the repository, so this is a
      courtesy check rather than a gate. Run it anyway; a reviewer may.
- [ ] If the branch touches any `.md`: `make docs-lint` passes. The
      `linting.yml` workflow calls the shared ROCm documentation linter
      (markdownlint plus a spell check). New proper nouns must be added to
      `.wordlist.txt` or the spell check fails.
- [ ] Generated artifacts are regenerated and committed, not hand-edited:
      `make generate` (deepcopy and mocks) and `make manifests` (CRDs, RBAC).
      A CRD change that is not regenerated will not match.

### 3. Tests, in upstream's style

The ROCm guide is explicit: **new functionality is only merged with new unit
tests**, existing tests must not break, and the pull request should include the
log of a successful test run.

Match the house style rather than importing this project's conventions:

- Table-driven tests using anonymous structs, as in `internal/utils_test.go`.
- `github.com/stretchr/testify/assert` is available and used. Plain `t.Errorf`
  with `reflect.DeepEqual` is equally common. Either is idiomatic here.
- `ginkgo`/`gomega` plus `go.uber.org/mock` are used for the controller and
  envtest suites (`internal/controllers`, `internal/kmmmodule`).
- Mocks are checked-in `mock_*.go` files produced by `make generate`. Do not
  hand-write one.
- No hardware in a unit test. Decision logic must be reachable without a GPU,
  and every test added by this fork must be able to fail — watch it go red
  against a deliberately broken implementation before trusting it.

#### The UNIT_TEST trap

The Makefile pins:

```make
UNIT_TEST ?= ./internal ./internal/controllers ./internal/kmmmodule
```

That is three exact packages, **not** `./internal/...`. Consequences:

- `internal/plugin/plugin_test.go` exists in upstream today and CI has never
  run it.
- A test added under `internal/metricsexporter`, `internal/nodelabeller`,
  `internal/plugin`, `internal/validator` or `internal/controllers/workermgr`
  is invisible to upstream CI.

**Therefore:** put new tests in one of the three covered packages where that is
natural, or extend `UNIT_TEST` in the same branch and say so in the pull request
body. A test the CI never runs is an assumption, not a test.

### 4. Pull request body

Use the org template's sections verbatim — Motivation, Technical Details, Test
Plan, Test Result, Submission Checklist. Populate all of them.

- [ ] **Motivation** — the user-visible problem. For APU work, link the ROCm
      issue that documents it rather than asserting it.
- [ ] **Technical Details** — what changed, per file, and why this shape.
- [ ] **Test Plan** — what was tested and how.
- [ ] **Test Result** — paste the actual `make unit-test` output. The guide asks
      for the log of a successful run.
- [ ] Submission checklist box ticked.
- [ ] Linked issue, if one exists upstream.

### 5. Prose and terminology

From `docs/contributing/documentation-standards.md`, which applies to prose in
pull requests and documentation alike:

- Active voice, second person, present tense.
- "AMD GPU Operator" — never "GPU operator" or "gpu-operator".
- "Kubernetes" — never "K8s".
- "AMD GPU driver" — never "AMDGPU driver" or bare "GPU driver".
- "DeviceConfig" — one word, capital D and C, when naming the resource.
- "worker node", not "worker" or "node" alone.
- "container image", not bare "image".
- Expand every acronym on first use per document: NFD (Node Feature Discovery),
  KMM (Kernel Module Management), CRD (Custom Resource Definition).

### 6. Code conventions

- [ ] Every new Go file carries the upstream licence header, copied byte for
      byte from a neighbouring file. Note that the AMD header contains literally
      escaped quotes — `the \"License\"` and `an \"AS IS\"`. Reproduce it exactly.
      Do not correct it; a diff that "fixes" it is noise in review.
- [ ] Commit subjects follow the prevailing conventional-commit style:
      `feat:`, `fix(kmm):`, `docs:`, `ci:`, `build:`. Not enforced, widely used.
- [ ] No internal ticket references. Upstream commits carry `GPUOP-`, `KUBE-`
      and `ROCM-` identifiers; those are AMD's tracker and mean nothing from
      outside.

### 7. House rules for this fork

These are ours, not upstream's, and they are not negotiable.

- [ ] Commits are authored by the human maintainer. Verify before the first
      commit on a branch:

      ```bash
      git -C ~/work/code/gpu-operator config user.name
      git -C ~/work/code/gpu-operator config user.email
      ```

- [ ] No tooling attribution anywhere: not in commit messages, branch names,
      code comments, issue bodies, or pull request bodies. No co-author
      trailers. (Upstream is inconsistent about this — pull request #465's body
      carries a generated-by line. That is not licence to match it.)
- [ ] **No hardware fact is guessed.** The Strix Halo PCI device identifier, its
      PCI class, and its KFD surface are read off the physical unit with
      `lspci -nn` and the sysfs tree. Until then they are marked pending, in a
      form that makes the omission obvious in review.
- [ ] **No claim of hardware validation that has not happened.** Not in a commit
      message, not in a pull request body, not in a code comment. If a change
      has only been reasoned about and unit-tested, the pull request says
      exactly that.
- [ ] No pull request is opened against `ROCm/gpu-operator` by anyone but the
      maintainer. Branches are pushed to this fork and stewarded from there.

## Branch hygiene

The fork's `main` tracks `upstream/main` exactly and must stay that way. Two
classes of branch live here:

| Prefix | Purpose | Base | Destination |
|---|---|---|---|
| `fork/` | Fork-only material, including this file | `upstream/main` | The fork's `main` only |
| `apu/` | Upstream-bound, one concern each | `upstream/main` | Offered to `ROCm/gpu-operator` |

Always cut from `upstream/main`, never from the fork's `main`, so that
fork-only commits cannot ride along in an upstream diff:

```bash
git fetch upstream
git checkout -b apu/<topic> upstream/main
```

Before offering a branch, confirm the diff contains only what you intend:

```bash
git diff --stat upstream/main...HEAD
```

## Pending hardware

The units have not arrived. Every item below is a fact to be read off the
physical hardware, not inferred from a specification sheet, and each blocks the
branch named beside it.

| Fact | How to obtain | Blocks |
|---|---|---|
| iGPU PCI device identifier | `lspci -nn -d 1002:` on the Halo unit | `apu/device-catalogue` |
| iGPU PCI class code | same command; confirm it is display class | `apu/nfd-igpu-detection` |
| KFD topology surface | `/sys/class/kfd/kfd/topology/nodes/*/properties` | exporter work |
| Which monitoring fields amdsmi answers | `amd-smi metric` on the unit | exporter work |

Until each is read, the corresponding branch carries the mechanism, the tests,
and an explicitly marked gap — never an invented value.

<!-- spellcheck-enable -->
