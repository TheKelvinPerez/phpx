# Context

## Terms

| Term | Meaning | Notes |
| --- | --- | --- |
| Project | One Composer project understood by Elefante. | A repository may contain several projects. |
| Repository | The version control root that contains one or more projects. | A project can also exist outside Git. |
| Composer root | The directory containing the selected `composer.json`. | This is the primary dependency boundary. |
| Application root | The directory from which framework commands should run. | It is often the Composer root, but Elefante keeps the concepts separate. |
| Workspace | One active checkout, branch checkout, or Git worktree of a repository. | Each workspace receives a stable local identity. |
| Environment | The runtime, provider, extensions, Composer executable, variables, processes, and local state used to execute a project workspace. | A workspace can be inspected before an environment is usable. |
| Provider | An adapter that inspects or prepares an execution topology and produces an executable process specification. | Phase 1 providers are compiled into the Elefante binary. |
| Native provider | The provider that uses compatible executables already available on the host. | It does not silently mutate system runtime selection. |
| Isolated provider | A provider that executes the project inside an isolated environment. | DDEV is the first isolated provider. |
| Facts | Read only observations derived from project files, the operating system, and provider inspection. | Facts never contain proposed mutations. |
| Requirement | A normalized PHP, extension, Composer, provider, or execution constraint with its source. | Original source text is preserved. |
| Plan | The deterministic ordered result of resolving facts, policy, provider observations, and a requested operation. | A plan is read only. |
| Action | One typed step inside a plan. | Actions declare effects, trust requirements, network requirements, and reversibility. |
| Mutation | An action that changes project files, provider state, machine state, cache state, or local Elefante state. | Mutations require approval according to policy. |
| Plan digest | A SHA256 digest of the canonical executable plan and its relevant inputs. | It prevents a reviewed plan from changing before execution. |
| Synchronization | Applying an approved plan so the selected environment satisfies project requirements and Composer dependencies are installed. | The command is `elefante sync`. |
| Trust approval | Local approval to execute Composer plugins or scripts for specific Composer input fingerprints. | Approval is never committed to the project. |
| Tool environment | An isolated cached Composer project used to execute a package binary without changing project dependencies. | The command is `elefante tool run`. |
| Human mode | The default terminal output and interaction contract. | Child output streams normally. |
| Machine protocol | The versioned newline delimited JSON event stream produced by `--json`. | Standard output contains protocol events only. |

## Relationships

| Relationship | Meaning |
| --- | --- |
| Repository contains Project | A repository may expose one or several Composer roots. |
| Workspace belongs to Repository | A branch checkout or Git worktree shares repository identity while retaining its own environment identity. |
| Environment belongs to Workspace and Project | Local state and mutations are isolated to the selected project inside the active workspace. |
| Discovery produces Facts | Discovery reads established files and machine state without executing project code. |
| Resolver produces Plan | Resolution combines facts, policy, provider observations, and the requested command. |
| Provider executes approved Actions | Providers do not choose product policy or render user output. |
| Plan digest approves Plan | A digest is valid only while all relevant inputs remain unchanged. |
| Composer owns package dependencies | Elefante invokes official Composer and does not replace its dependency solver. |
