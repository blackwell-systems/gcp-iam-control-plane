# Monetization Principles

> **Last updated:** 2025-01-28  
> **Status:** Maintainer commitment  
> **Changes require:** Public discussion and maintainer consensus

---

## The Core Commitment

**This project will never gate capabilities that currently exist in open source.**

If a feature is available today (2025-01-28) in the Blackwell Systems GCP emulator ecosystem, it will remain free and open source forever.

### Definition: Existing Capability

An "existing capability" is defined as:
- **Functionality that is implemented, documented, and usable** in a released version
- **Behavior that users reasonably rely on** for correctness or enforcement

**Planned features, undocumented internals, experimental branches, or implied use cases do not qualify as existing capabilities.**

This definition prevents bad-faith arguments about "implied features" while protecting genuine user expectations.

---

## What Stays OSS Forever

The following capabilities are **permanently open source** under Apache 2.0:

### Core Infrastructure
✅ **IAM policy engine** - Complete policy evaluation logic  
✅ **Enforcement proxy** (gcp-emulator-auth) - Pre-flight authorization  
✅ **All enforcement modes** - Off, Permissive (fail-open), Strict (fail-closed)  
✅ **Service emulators** - Secret Manager, KMS, and all future data plane emulators  
✅ **CLI orchestration** - Start, stop, status, logs commands  
✅ **Raw observability** - Logs, traces, error messages, failure reasons  

### Key Guarantees
- **Full enforcement** - All IAM modes remain fully functional
- **Production semantics** - Real permission checking, not mocked
- **No feature regression** - Existing capabilities never become paid-only
- **No artificial limits** - No request caps, user limits, or time restrictions

### Why These Stay Free

These are **infrastructure primitives**. They define:
- **Correctness** - Does the system work properly?
- **Trust** - Can teams rely on this for production validation?
- **Adoption** - Does the ecosystem grow?

Taking these away would:
- Invite forks and fragmentation
- Destroy community trust
- Eliminate the foundation for commercial offerings

**The infrastructure is the moat. We monetize workflows built on top, not the infrastructure itself.**

---

## Where Monetization Lives

Premium offerings are **additive workflow tools** that sit above the OSS core. They help teams do more with the infrastructure, not replace it.

### Premium Capabilities (Paid)

#### 1. Policy Intelligence
**What:** Automated policy generation from emulator traces

**OSS today:**
```bash
gcp-emulator logs iam
# Raw text logs, manual analysis required
```

**Premium:**
```bash
gcp-emulator-pro policy generate --from-trace --format=terraform
# Auto-generates least-privilege Terraform/YAML policies
```

**Why premium:**
- Requires aggregation and analysis over time
- Opinionated policy optimization algorithms
- Structured infrastructure-as-code output
- Saves hours of manual IAM policy writing

**Net-new capability:** This didn't exist in OSS. We're not taking anything away.

---

#### 2. Compliance & Audit Reports
**What:** Structured security reports for CI/CD and compliance teams

**OSS today:**
```bash
gcp-emulator logs iam
# Human-readable logs
```

**Premium:**
```bash
gcp-emulator-pro compliance report --format=json
# SOC2-friendly structured reports with recommendations
```

**Why premium:**
- Requires structured data aggregation
- Security analysis and recommendations
- CI/CD-ready artifacts (JSON, SARIF, etc.)
- Audit-trail metadata (who, what, when, why)

**Net-new capability:** Raw logs remain free. Structured compliance artifacts are premium.

---

#### 3. Centralized Policy Management (SaaS)
**What:** Hosted service for team policy coordination

**Features:**
- Web UI for policy editing and collaboration
- Centralized policy versioning and rollback
- Team synchronization across developers
- Policy change audit logs
- Usage analytics and insights

**Why premium:**
- Requires cloud infrastructure (hosting costs)
- Multi-tenancy, authentication, backups
- Ongoing operational support
- Network effects (team collaboration features)

**Net-new capability:** Local emulators remain fully functional. Central coordination is optional premium service.

---

#### 4. Enterprise Support & Services
**What:** Professional services for teams adopting the control plane

**Offerings:**
- Onboarding and training
- Policy reviews and migration
- CI/CD pipeline integration
- Custom emulator development
- Priority bug fixes and feature requests

**Why premium:**
- Requires dedicated engineering time
- Custom consulting work
- SLA guarantees

**Net-new capability:** Community support remains free. Priority access is premium.

---

## The Decision Framework

Before considering any feature for premium tier, apply this test:

### The Bright Line Test

> **"If we make this premium-only, are we taking away something users already have?"**

**If YES** → Feature must stay OSS. Find a different approach.  
**If NO** → Feature can be premium if it's a net-new capability.

### Examples

| Feature | OSS or Pro? | Reasoning |
|---------|-------------|-----------|
| Strict mode enforcement | **OSS** | Already exists, core correctness |
| Generate Terraform from trace | **Pro** | New capability, didn't exist before |
| Raw trace logs | **OSS** | Already exists, core observability |
| Compliance JSON reports | **Pro** | Structured analysis is net-new |
| Service emulators (SM, KMS) | **OSS** | Already exist, core infrastructure |
| Centralized policy sync (SaaS) | **Pro** | New service, costs infrastructure |
| Custom roles | **OSS** | Already supported, core feature |
| Policy recommendations | **Pro** | Intelligent analysis is net-new |
| All three modes (off/perm/strict) | **OSS** | Already exist, core correctness |
| Web dashboard UI | **Pro** | New interface, optional convenience |

---

## Architectural Separation

Premium features are built **as a layer above** the OSS core, not as gates within it.

```
┌─────────────────────────────────────────────────┐
│  PREMIUM LAYER (Closed Source)                  │
│                                                  │
│  ┌──────────────────┐  ┌─────────────────────┐ │
│  │ Policy Stream    │  │ Compliance Reports  │ │
│  │ Dashboard        │  │ Generator           │ │
│  └──────────────────┘  └─────────────────────┘ │
│                                                  │
│  ┌──────────────────────────────────────────┐  │
│  │ Centralized Policy Management (SaaS)     │  │
│  └──────────────────────────────────────────┘  │
│                                                  │
│  Imports OSS libraries, adds workflow tools     │
└────────────────┬────────────────────────────────┘
                 │
                 │ Uses as library
                 │
┌────────────────▼────────────────────────────────┐
│  OSS LAYER (Apache 2.0 - Forever Free)          │
│                                                  │
│  ├── gcp-iam-emulator (policy engine)           │
│  ├── gcp-emulator-auth (enforcement proxy)      │
│  ├── gcp-secret-manager-emulator                │
│  ├── gcp-kms-emulator                           │
│  └── gcp-iam-control-plane (CLI)                │
│                                                  │
│  Complete enforcement, all modes, full control  │
└─────────────────────────────────────────────────┘
```

**Key principle:** Premium code imports OSS libraries. It never modifies them or gates their functionality.

---

## What We Will NOT Do

The following practices are explicitly forbidden:

❌ **Gate existing enforcement modes** - Strict mode will never become paid-only  
❌ **Add license checks to hot paths** - No runtime validation in enforcement logic  
❌ **Cripple OSS capabilities** - No artificial limits on requests, users, or time  
❌ **Relicense existing repos** - Apache 2.0 is permanent for current code  
❌ **Create "community vs pro" forks** - One codebase for OSS, separate repos for premium  
❌ **Take away features retroactively** - If it's free today, it's free forever  

**Why these are forbidden:**

These tactics invite:
- Community backlash and loss of trust
- Project forks that fragment the ecosystem
- Reputation damage that kills both OSS and commercial efforts
- Competitive disadvantage (forks become competitors)

**We don't need these tactics.** There is ample commercial value in workflow tools without compromising the infrastructure layer.

---

## The Strategic Rationale

### Why This Model Works

**1. Trust is our moat**
- Open infrastructure attracts users
- Users become contributors
- Contributors become advocates
- Advocates become customers

**2. Infrastructure wins adoption, products monetize workflows**
- OSS = "Does it work correctly?" (adoption driver)
- Pro = "How do we use this at scale?" (monetization driver)

**3. Premium tools require the OSS foundation**
- Policy generation needs traces (OSS provides traces)
- Compliance reports need enforcement (OSS provides enforcement)
- Central management needs local enforcement (OSS provides local)

**The OSS layer is not in competition with premium - it's the prerequisite.**

### Proven Market Validation

**LocalStack** (AWS emulator suite):
- OSS: Basic emulators, permissive by default
- Pro: $39-89/user/month for IAM enforcement, compliance, persistence
- Validates the open-core + workflow monetization model at scale

**We have the same opportunity with GCP:**
- Google Cloud: 13% market share, $10B+ quarterly revenue
- No official IAM enforcement for emulators
- Enterprise teams need pre-deployment security validation

---

## Governance

### This Document Is a Maintainer Commitment

This document represents a **commitment to users and the community**. Changes require:

1. **Public discussion** - Proposed changes must be announced via GitHub Discussion
2. **Community input period** - Minimum 30 days for feedback
3. **Transparent rationale** - Clear explanation of why changes are needed
4. **Maintainer consensus** - Active maintainers must agree by majority vote

**Note:** Maintainers retain final authority to ensure project sustainability. However, changes that violate the core commitment (gating existing features) would fundamentally break trust and are not permissible.

### Enforcement

If this commitment is violated:
- Community has full rights to fork under Apache 2.0
- Maintainers have right to revert changes
- This document can be cited in any dispute

### Updates

Minor clarifications (typos, examples, formatting) do not require community input.

Substantive changes (moving features between OSS/Pro, changing principles) require full governance process.

---

## Long-Term Vision

### 5-Year Goal
- **OSS:** Industry-standard local IAM control plane for GCP
- **Pro:** Premium workflow tools serving hundreds of enterprise teams
- **Ecosystem:** Thriving community of contributors and commercial customers

### Success Metrics
- **OSS adoption:** 10,000+ GitHub stars, active contributor community
- **Commercial traction:** Paying customers using premium features
- **Trust maintained:** Zero controversial feature gating, strong reputation

---

## Summary

**The Rule:**
> OSS = correctness & enforcement  
> Pro = insight, reporting, coordination, scale

**The Commitment:**
> Never take away what users already have.  
> Monetization must be additive, not subtractive.

**The Result:**
> A sustainable business built on trust, not extraction.

---

## Questions?

- **"Will strict mode ever become paid-only?"** No. Never. It's core infrastructure.
- **"Can I fork this if you violate these principles?"** Yes. Apache 2.0 guarantees it.
- **"What if a premium feature becomes table stakes?"** We'll evaluate moving it to OSS after community discussion.
- **"How do you make money without gating features?"** We charge for workflow tools (policy generation, compliance reports, SaaS) that save time and improve visibility.

---

**This is how we build a company without burning the community.**

## Contact

- **Community Discussion:** [GitHub Discussions](https://github.com/blackwell-systems/gcp-emulator-auth/discussions)
- **Commercial Inquiries:** sales@blackwell.systems
- **Security Issues:** security@blackwell.systems

---

© 2025 Blackwell Systems. This document is licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
