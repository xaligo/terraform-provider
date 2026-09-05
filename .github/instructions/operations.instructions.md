---
applyTo: "**"
---

# AI development operations

Use this minimal workflow:

```bash
# 1. Route the task and load its preconditions.
sed -n '1,240p' .github/instructions/index.instructions.md
sed -n '1,320p' .github/instructions/01-general/01-01-general-project-preconditions.instructions.md

# 2. Establish state without overwriting unrelated work.
git status --short
git diff -- <in-scope-paths>
git diff --cached -- <in-scope-paths>

# 3. Locate before reading, then make the smallest coherent change.
rg -n '<symbol|resource-type|diagnostic>' <likely-paths>

# 4. Run checks appropriate to the changed scope once they exist.
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
git diff --check

# 5. Validate Terraform examples and generated fixtures when applicable.
terraform fmt -check -recursive samples
xaligo validate <generated-fixture.xal>

# 6. Run acceptance tests only through the explicit acceptance-test gate.
TF_ACC=1 go test ./test/internal/controller/... -run 'TestAcc'

# 7. Audit the handoff.
git status --short
git diff --check
```

Ordinary unit tests must not run Terraform or require provider downloads.
Acceptance tests may run a local Terraform CLI only when explicitly selected
with `TF_ACC`; they use isolated temporary directories, local state, and no
cloud credentials or remote services. Never point tests at a user's working
directory or backend. Do not commit generated binaries, caches, `.terraform`
directories, state files, plans, credentials, or rendered output.

Do not create commits unless the user asks for them. Never push, publish, tag,
open a pull request, or rewrite history without explicit authorization.
