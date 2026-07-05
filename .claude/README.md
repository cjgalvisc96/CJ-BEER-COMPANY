# Claude Code Harness

Configures AI-assisted development for CJ-BEER-COMPANY: hard rules, reusable
skills, and runnable commands, so an agent contributes without re-deriving
the architecture each session.

```
.claude/
├── rules/
│   ├── architecture.md     # layering, context isolation, event contracts
│   ├── coding-style.md     # Go idioms, SOLID/DRY/KISS/YAGNI as applied here
│   └── testing.md          # pyramid, fakes-over-mocks, race detector
├── skills/
│   ├── add-bounded-context/SKILL.md
│   └── add-use-case/SKILL.md
└── commands/
    └── quality-gate.md     # the pre-merge check (lint + race tests)
```

**Rules** are always-on constraints — read them before changing code.
**Skills** are step-by-step procedures for recurring structural work.
**Commands** are runnable workflows mirroring CI.
