# OSS README & Marketing Research — 2026-04-16

Research on making open source projects successful, with specific focus on README best practices, developer adoption patterns, and competitive positioning for freetown (three-ledger accounting library).

---

## 1. What Developers Actually Do (Survey Data)

### Catchy Agency Survey (202 developers, All Things Open 2025)
- **73% want hands-on experience within minutes** — quickstart matters more than pitch
- **Documentation is #1 trust signal (34.2%)** AND #1 abandonment trigger (17.3%) — single highest-leverage surface
- **26.2% abandon if project looks unmaintained** — visible activity signals matter independent of code quality
- **12.4% influenced by well-known users** — social proof works
- One-command install or <5 min setup is the benchmark

### GitHub Open Source Survey (5,500 respondents, 2017)
- **72% always seek out open source options** when evaluating tools
- **93% complained docs are incomplete or outdated** — #1 annoyance
- **86% said security was extremely/very important**
- **88% said stability was extremely important**
- **67% said license is a determinant** when deciding to contribute

### Time to First Hello World (TTFHW)
- Measures time from first encounter to first working example
- Low TTFHW correlates with higher retention and conversion
- High TTFHW signals confusing docs, complex setup, or missing sandbox environments

### Redpoint Ventures Data (80 developer tool companies)
- Median stars at Seed: 2,850 / at Series A: 4,980
- Average MoM star growth: 7.9%
- Median contributors at Seed: 25 / at Series A: 50
- "We view early-stage open source companies like social networks — community momentum takes precedence over revenue"

---

## 2. README Structure Consensus

Across Tom Preston-Werner (README-driven development), makeareadme.com, Dan Bader's guide, awesome-readme, and the exemplar analysis:

1. Project name + **one-line description that names the problem**
2. Badges (build status, version, license)
3. Hero image / animated GIF / screenshot
4. Quick-start / installation (copyable code block)
5. Usage examples (progressive: simple → realistic → advanced)
6. Features
7. API / configuration reference
8. Contributing guide
9. License
10. Credits / acknowledgments

### "Steal These Patterns" Checklist (from exemplar repos)
- **Progression:** hello world → realistic → advanced composition (clear prerequisites)
- **One folder per use-case:** each example runnable in isolation (README, config, expected output)
- **Cross-links:** each recipe links back to primitives it uses; primitives link forward to recipes
- **Testing:** examples are CI-verified (executed, compiled, or snapshot-tested)
- **Consistency:** same naming conventions, same structure, same "shape" across examples

---

## 3. Advice from OSS Founders

### Tom Preston-Werner (GitHub) — README Driven Development
- "Write your README first. Before you write any code or tests or behaviors or stories or ANYTHING."
- "A perfect implementation of the wrong specification is worthless. A beautifully crafted library with no documentation is also nearly worthless."
- The README is "the single most important document in your codebase."
- Writing it retroactively is "an absolute drag" — write it when excitement is highest.

### Mitchell Hashimoto (HashiCorp/Terraform)
- **Performance-based marketing**: Set up timers showing how fast HashiCorp shipped support for new cloud features vs. the clouds themselves. Directly countered the #1 objection.
- Founders must personally understand the GTM motion before scaling it.
- Community onboarding matters as much as employee onboarding.
- "While Armon onboarded every employee, Mitchell onboarded the community."

### Guillermo Rauch (Vercel/Next.js)
- **"Be humble in early language"** — don't claim category leadership before earning it. Be specific about what you solve.
- Start with a framework/tool, not a product. The commercial product comes later.
- **Narrow scope ruthlessly** — PMF came when they narrowed to "the best React experience."
- Learn from your most-loved feature, not your product vision.

### Lago — How They Got First 1000 Stars
- Published 2 articles/week consistently
- Spent as much time distributing content as producing it
- Out of ~60 articles, only 3-4 got traction on HN — persistence despite low hit rate
- 6 months to 1,000 stars, then only 14 days to 1,500 (compounding)
- Every self-hosted signup triggered a welcome email inviting users to star the repo

### PHPStan — 0 to 1,000 Stars in Three Months
- "Start with implementing the core idea where the value lies"
- "Release the first version as soon as it's useful. Don't wait for it to be perfect."
- "Without marketing, even the best projects would starve."

### ToolJet — Zero to 10,000 Stars
- Launched on Product Hunt first, then HN a few hours later
- **Ported server from Ruby to JavaScript** because two languages was a contributor barrier
- Regular blog posts about technology and best practices attracted both traffic and stars

---

## 4. Launch Platform Timing
- **Hacker News:** 8:00-10:00 AM PT, Tuesday-Thursday
- **Product Hunt:** start at 12:01 AM PT for full 24-hour voting cycle
- **Reddit:** ~30 minutes after HN post (stagger to manage feedback)
- Respond to comments within 2 hours — boosts visibility in algorithms
- For libraries/CLI tools, HN and Reddit outperform Product Hunt. For products with a UI, Product Hunt is stronger.

---

## 5. Competitive Landscape: Billing/Accounting OSS

### Billing Platforms (full applications, not libraries)

| Project | One-liner | README strength | README weakness |
|---------|-----------|-----------------|-----------------|
| **Lago** | "The AI-native billing platform" | Social proof first (customer logos), features scannable, clear cloud vs self-hosted | "AI-native" rebrand obscures what it does |
| **Kill Bill** | "Open-Source Subscription Billing & Payments Platform" | 15+ year track record, enterprise credibility | Corporate/vague, no code, no teaching |
| **Flexprice** | "Monetization Infrastructure Built for AI Native Companies" | Sharpest problem articulation — names 4 explicit pain points | New/unproven, "AI Native" trend-chasing |
| **OpenMeter** | "Open-source metering and billing platform" | Architecture section is a differentiator, SDK table clean | "AI, agentic and DevTool" feels like keyword stuffing |
| **Lotus** (defunct) | "Pricing & Packaging Infrastructure For Any Business Model" | Best problem framing in the space — named the business problem | Tech stack details too prominent |

### Fintech Ledger Infrastructure (closest architectural comparables)

| Project | One-liner | README strength | README weakness |
|---------|-----------|-----------------|-----------------|
| **TigerBeetle** | "The financial transactions database" | Boldest positioning ("next 30 years"), working REPL immediately | Cryptically minimal — won't learn what it is from README |
| **Formance Ledger** | "A programmable financial core ledger" | Use-case-driven positioning, numscript DSL differentiator | Too broad, assumes double-entry knowledge |
| **Blnk Finance** | "Open-Source Financial Ledger for Developers" | "Fast without compromising compliance and correctness" — clean tension | Very thin README, no code examples |

### Accounting Libraries (closest category match)

| Project | One-liner | README strength | README weakness |
|---------|-----------|-----------------|-----------------|
| **Medici** (Node.js) | "Double-entry accounting system for nodejs + mongoose" | Best technical README — teaches through code, performance section | No problem statement, no "why" |
| **GoDBLedger** (Go) | "Make double entry bookkeeping transactions programmable" | Multiple integration paths (gRPC, CLI, files) | Server not library, messy/sprawling README |
| **ACCCORE** (Go) | "A core accounting library made in golang" | Narrow focus (virtual currencies) | Barely a README, no code examples |
| **DEB** (Go) | "Double-entry bookkeeping library" | — | Essentially no README, abandoned |

### Plain Text Accounting

| Project | One-liner | Notable |
|---------|-----------|---------|
| **hledger** | "Robust, friendly, fast, plain text accounting" | "Any countable commodity" framing is broad and interesting |
| **Beancount** | "Double-Entry Accounting from Text Files" | "Largely does away with credits and debits" — opposite design choice from freetown |

---

## 6. Positioning Analysis

### The gap freetown occupies

Nobody occupies "accounting library for monetization." The landscape splits cleanly:
- **Billing platforms** (Lago, Kill Bill, OpenMeter, Flexprice) — full applications with metering, invoicing, payment orchestration
- **Fintech ledgers** (TigerBeetle, Formance, Blnk) — infrastructure for moving money, focused on performance/safety
- **Accounting libraries** (Medici, GoDBLedger, ACCCORE) — generic double-entry bookkeeping, no monetization domain knowledge
- **Personal accounting** (hledger, Beancount) — individual finance, text files

Freetown is a **domain-aware accounting library** — you embed it, not deploy it. This is an unclaimed position.

### The Go accounting library space is dead
DEB, ACCCORE, go-accounting are all minimal/abandoned. GoDBLedger is active but it's a server, not a library. Freetown would be the only serious Go accounting library with active development.

### "AI-native" is the current bandwagon
Lago, OpenMeter, Flexprice, and Amberflo have all recently added "AI" to their positioning. Crowded and increasingly meaningless. Avoid this trap.

### Nobody explains accounting well
Medici comes closest by teaching through code. Beancount has real educational content. But most projects assume you know double-entry or don't care. The three-ledger model is genuinely novel and needs to be explained, not assumed.

### Library vs. platform is the sharpest edge
Every billing project is a platform you deploy. Every accounting library is domain-ignorant. Freetown is a *library* with *domain knowledge* — you embed it, you don't deploy it.

---

## 7. What the Current README Gets Wrong

Against all of the above, the current freetown README:
- **No problem statement** — jumps straight into "4-stage pipeline" architecture
- **No "who is this for"** — reader can't self-select
- **Internal jargon** (CreditService, JournalRepository, GL concepts) before explaining what it does
- **No one-liner** that names the problem or value
- **Import path is `stunning-octo-lamp`** — placeholder, not a real project identity
- **"Coming soon" section** signals incompleteness before establishing value
- **References internal docs** (ADRs, arch work) that a public visitor can't access

---

## 8. Best Patterns to Steal

From the exemplar analysis, the patterns that best match freetown's position:

1. **Flexprice's problem articulation** — name explicit pain points the developer recognizes
2. **Medici's code-first teaching** — teach the three-ledger model through usage, not theory
3. **Formance's use-case framing** — "if you're building X, this is for you"
4. **Lago's social proof positioning** — once there are users, lead with them
5. **TigerBeetle's ambition** — make a bold claim about what you're building toward (but only once you can back it up)

### Anti-patterns to avoid
- Corporate vagueness (Kill Bill)
- Feature-dumping without explaining "why" (OpenMeter)
- "We do everything" positioning (Lago's recent shift)
- Trend-chasing taglines ("AI-native")
- No problem statement (Medici, TigerBeetle, ACCCORE)
- Assuming the reader already knows they need double-entry bookkeeping

---

## Sources

### Surveys and Data
- Catchy Agency: What 202 Open Source Developers Taught Us About Tool Adoption
- GitHub Open Source Survey 2017 (opensourcesurvey.org/2017/)
- Tidelift 2024 State of the Open Source Maintainer Report
- Redpoint Ventures: How Many Stars Is Enough?

### Guides and Frameworks
- Tom Preston-Werner: Readme Driven Development (tom.preston-werner.com)
- Dan Bader: How to Write a Great README (dbader.org)
- makeareadme.com
- awesome-readme (github.com/matiassingers/awesome-readme)
- Adam Stacoviak: Top Ten Reasons I Won't Use Your OSS Project (Changelog)
- TODO Group: Marketing Open Source Projects
- GitHub Blog: Marketing for Maintainers
- GitHub Open Source Guides: Finding Users (opensource.guide/finding-users/)

### Founder Lessons
- Mitchell Hashimoto lessons (antoinebuteau.com, Heavybit talk)
- Guillermo Rauch / Vercel PMF (First Round Review)
- Jeff Lawson / Twilio: Ask Your Developer

### Case Studies
- Lago: How We Got Our First 1000 GitHub Stars
- ToolJet: Zero to 10,000 Stargazers
- PHPStan: 0 to 1,000 Stars in Three Months
- daily.dev: Step-by-Step Launch Guide

### DX Metrics
- APIscene: Time to Hello World and Developer LTV
- Moesif: Developer Experience Metrics That Matter
- Nordic APIs: Why Time to First Call Is a Vital API Metric
