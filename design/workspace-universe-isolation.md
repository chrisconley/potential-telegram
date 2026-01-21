# Workspace × Universe: Two-Dimensional Isolation Model

**Date:** 2025-12-05
**Status:** Accepted
**Context:** Defining the isolation boundaries for events, schemas, and customer identity

---

## Summary

We've implemented a **two-dimensional isolation model** for the metering platform:

```
(WorkspaceID, Universe) → Event isolation boundary
```

- **WorkspaceID**: Operational dimension - regions, divisions, business units
- **Universe**: Data/namespace dimension - production, test, simulation, legacy systems

These dimensions are **orthogonal** and form a **many-to-many relationship**:
- Multiple workspaces can share a universe (global customers across regions)
- A workspace can participate in multiple universes (production + test + simulations)
- Events and customer identity are scoped to this intersection

---

## Context & Problem

### The Challenge

We're building a metering platform that needs to support:

1. **Multi-region deployments** where the same customer exists across regions
2. **Test/staging/production isolation** without duplicating all configuration
3. **Post-merger scenarios** where legacy systems need gradual migration
4. **What-if simulations** for scenario planning and forecasting
5. **Operational boundaries** where different regions/BUs have different event schemas

### Initial Single-Dimension Approach

Our initial design had a single `TenantID` field that tried to represent both:
- **Operational boundaries** (which region/BU processes this event)
- **Data boundaries** (which customer namespace does this belong to)

This created conflicts:

**Problem 1: Multi-region with global customers**
```
Scenario: Acme Corp has customers in US, EU, and APAC
- US events need different schemas than EU events (regulatory differences)
- But customer:123 should be the same customer globally
- Single TenantID forces us to choose: duplicate customers OR duplicate schemas
```

**Problem 2: Test data isolation**
```
Scenario: Testing new billing rules
- Need realistic production data for testing
- Can't mix test transactions with production for customer:123
- Single TenantID means: test-tenant vs prod-tenant = different customer IDs
```

**Problem 3: Post-merger integration**
```
Scenario: Acme acquires WidgetCo
- WidgetCo has customer:123 in their system
- Acme has customer:123 in their system
- These are different customers, but IDs collide
- Need namespace separation during migration
```

**Problem 4: Simulation and forecasting**
```
Scenario: "What if we change pricing on Q4 data?"
- Need to replay Q4 production events with new rules
- Can't affect production customer:123's actual billing
- Need parallel reality where customer:123 in simulation ≠ customer:123 in production
```

### The Core Insight

These problems arise because we're conflating two orthogonal concerns:

1. **Configuration/Operational concern**: "Where is this event processed and how is it shaped?"
   - Different regions have different schemas (GDPR in EU, different regulations in US)
   - Different business units have different event types
   - This is about **operational boundaries** and **event schemas**

2. **Data/Namespace concern**: "Which reality/timeline does this data belong to?"
   - Production vs test vs simulation
   - Legacy system A vs legacy system B during merger
   - This is about **customer identity** and **data isolation**

**These concerns are independent**:
- Events from US and EU workspaces can both belong to "production" universe (same customers)
- Events from a single workspace can belong to "production", "test", or "simulation" universes (different customer namespaces)

---

## Decision

### Two-Dimensional Model

We separate these concerns into two explicit, orthogonal dimensions:

```
┌─────────────┬──────────────────────────────────────────────────────┐
│ Dimension   │ Purpose                                              │
├─────────────┼──────────────────────────────────────────────────────┤
│ WorkspaceID │ Operational boundary                                 │
│             │ - Region, division, business unit                    │
│             │ - Owns event schemas                                 │
│             │ - Different workspaces can have event types with     │
│             │   the same name but different schemas                │
│             │                                                      │
│ Universe    │ Data/customer namespace boundary                     │
│             │ - production, test, staging, simulation              │
│             │ - Scopes customer identity                           │
│             │ - customer:123 in "production" universe ≠           │
│             │   customer:123 in "test" universe                    │
└─────────────┴──────────────────────────────────────────────────────┘
```

### Schema Lookup Pattern

**Event schemas** are looked up by `(WorkspaceID, EventType)`:
```
(WorkspaceID, EventType) → Event Schema
```

- Workspace **owns** the event schema
- Different workspaces can have event types with the same name but completely different schemas
- Example:
  - `(acme-us, "api.request")` → schema with US-specific fields
  - `(acme-eu, "api.request")` → schema with EU-specific fields (GDPR extras)

### Customer Identity Scoping

**Customer identity** is scoped by `(Universe, Subject)`:
```
(Universe, Subject) → Unique Customer
```

- Same `Subject` in different universes = different customers
- Example:
  - `("production", "customer:123")` → production customer 123
  - `("test", "customer:123")` → test customer 123 (different entity)
  - `("simulation-q4", "customer:123")` → simulated customer 123 (different entity)

### Many-to-Many Relationship

Workspaces and Universes intersect in a **many-to-many** relationship:

**Multiple workspaces can share a universe:**
```
("production" universe)
  ← acme-us workspace (US events, production customers)
  ← acme-eu workspace (EU events, production customers)
  ← acme-apac workspace (APAC events, production customers)

Result: Global customers exist across all regional workspaces
```

**A workspace can participate in multiple universes:**
```
(acme-us workspace)
  → "production" universe (live billing)
  → "test" universe (QA environment)
  → "simulation-q4-pricing" universe (what-if scenarios)

Result: Same event processing logic, different customer namespaces
```

### Event Data Model

```go
type EventPayload struct {
    TransactionID EventPayloadTransactionID
    WorkspaceID   EventPayloadWorkspaceID    // Operational boundary
    Universe      EventPayloadUniverse        // Data/namespace boundary
    EventType     EventPayloadType
    Subject       EventPayloadSubject           // Scoped to Universe
    Timestamp     EventPayloadTimestamp
    Properties    EventPayloadProperties
}
```

```go
type Event struct {
    TransactionID EventTransactionID
    WorkspaceID   EventWorkspaceID    // Operational boundary
    Universe      EventUniverse        // Data/namespace boundary
    EventType     EventType
    Subject       EventSubject           // Scoped to Universe
    Timestamp     EventTimestamp
    Measures      EventMeasures        // Typed by workspace schema
    Dimensions    EventDimensions      // Typed by workspace schema
}
```

---

## Use Cases Solved

### 1. Multi-Region with Global Customers

**Scenario**: Acme Corp operates in US, EU, and APAC with global customers.

**Solution**:
```
Workspaces: acme-us, acme-eu, acme-apac (different schemas)
Universe: "production" (shared customer namespace)

Events:
- (acme-us, "production", "customer:123", "api.request", {...})
- (acme-eu, "production", "customer:123", "api.request", {...})
  → Same customer:123, different schemas, unified billing
```

**Benefits**:
- Regional event schemas can differ (regulatory compliance)
- Customer identity is global (no duplication)
- Billing aggregates across all regions for customer:123

### 2. Test/Staging/Production Isolation

**Scenario**: Need to test new billing rules without affecting production.

**Solution**:
```
Workspace: acme-us (same event schemas)
Universes: "production", "test", "staging" (isolated namespaces)

Events:
- (acme-us, "production", "customer:123", ...)  → production billing
- (acme-us, "test", "customer:123", ...)        → test environment
  → Different customer:123 entities, same schemas
```

**Benefits**:
- Test data completely isolated from production
- Same event processing logic across environments
- Can use production-like customer IDs in test without collision

### 3. Post-Merger Integration

**Scenario**: Acme acquires WidgetCo; both have customer:123.

**Solution**:
```
Phase 1: Separate universes during migration
- (acme-us, "legacy-acme", "customer:123", ...)
- (widgetco, "legacy-widget", "customer:123", ...)
  → Different customers, no ID collision

Phase 2: Gradual migration to shared universe
- Map legacy-widget customers → production universe with new IDs
- Or keep universe separation for customer choice
```

**Benefits**:
- No customer ID collisions during merger
- Gradual migration without big-bang cutover
- Can maintain separate billing until integration complete

### 4. What-If Simulations and Forecasting

**Scenario**: "What if we change pricing rules and replay Q4 data?"

**Solution**:
```
Workspace: acme-us (same schemas)
Universes: "production", "simulation-q4-new-pricing"

Process:
1. Copy Q4 events from production to simulation universe
2. Apply new pricing rules to simulation
3. Compare results without affecting production
```

**Benefits**:
- Risk-free scenario testing
- Same customer IDs in simulation and production (easy comparison)
- Multiple simulations can run in parallel (different universes)

### 5. Legacy System Migration

**Scenario**: Migrating from multiple legacy billing systems to unified platform.

**Solution**:
```
Phase 1: All legacy systems in separate universes
- (legacy-system-a, "legacy-a", "customer:123", ...)
- (legacy-system-b, "legacy-b", "customer:123", ...)

Phase 2: Gradual migration
- Move customers to (unified, "production", "customer:NEW_ID", ...)
- Keep legacy universes for historical data

Phase 3: Consolidate (optional)
- Merge universes once migration complete
```

**Benefits**:
- Parallel operation of old and new systems
- Gradual customer migration
- Historical data preserved with clear lineage

---

## Industry Precedent

Our Workspace × Universe model has strong precedent in production systems. We researched industry leaders and found similar two-dimensional isolation patterns:

### Snowflake: Account × Database/Share

**Model**: Account (operational) × Database/Share (data universe)
- **Account**: Compute, RBAC, network policies, regional boundary
- **Database/Share**: Logical data universe with shared access
- **Many-to-many**: Multiple accounts can import same share; an account can import many shares

**Use cases**: Multi-region data sharing, SaaS multi-tenant analytics, post-merger data sharing

**Our alignment**: WorkspaceID ≈ Account, Universe ≈ Database/Share

**Sources**:
- [Snowflake Secure Data Sharing](https://docs.snowflake.com/en/user-guide/data-sharing-intro)
- [Cross-region sharing](https://docs.snowflake.com/en/user-guide/secure-data-sharing-across-regions-platforms)

### Databricks Unity Catalog: Workspace × Metastore/Catalog

**Model**: Workspace (compute/ops) × Metastore/Catalog (data governance)
- **Workspace**: Notebooks, jobs, clusters, permissions
- **Metastore/Catalog**: Data objects, governance, shared across workspaces
- **Many-to-many**: Multiple workspaces share one metastore; workspace consumes multiple catalogs via Delta Sharing

**Use cases**: Multi-workspace shared data platform, cross-region data mesh, M&A scenarios

**Our alignment**: WorkspaceID ≈ Workspace, Universe ≈ Catalog/Metastore

**Sources**:
- [Unity Catalog Overview](https://docs.databricks.com/aws/en/data-governance/unity-catalog/)
- [Cross-metastore sharing](https://medium.com/databricks-unity-catalog-sme/a-practical-guide-to-catalog-layout-data-sharing-and-distribution-with-databricks-unity-catalog-f34fa822a367)

### Google BigQuery: Project × Dataset

**Model**: Project (billing/ops) × Dataset (data container)
- **Project**: Billing, IAM, API enablement, compute reservations
- **Dataset**: Top-level container for tables, often per-tenant or per-domain
- **Many-to-many**: A project can query datasets from many projects; a dataset can be accessed by many projects

**Use cases**: Multi-tenant SaaS analytics, multi-org data sharing, multi-region with replication

**Our alignment**: WorkspaceID ≈ Compute Project, Universe ≈ Dataset Project

**Sources**:
- [BigQuery multi-tenant best practices](https://cloud.google.com/bigquery/docs/best-practices-multi-tenancy)

### AWS Lake Formation: Account × Database/Table

**Model**: AWS Account (security boundary) × Lake Formation Database (data universe)
- **Account**: Security, billing, organizational boundary
- **Database/Table**: Data catalog resources shared across accounts
- **Many-to-many**: Databases shared with multiple accounts/OUs; accounts consume multiple shared databases

**Use cases**: Multi-tenant data lake, cross-account analytics, external partner data sharing

**Our alignment**: WorkspaceID ≈ AWS Account, Universe ≈ Lake Formation Database

**Sources**:
- [AWS multi-account strategy](https://docs.aws.amazon.com/whitepapers/latest/organizing-your-aws-environment/organizing-your-aws-environment.html)
- [Lake Formation cross-account access](https://aws.amazon.com/blogs/big-data/design-patterns-for-an-enterprise-data-lake-using-aws-lake-formation-cross-account-access/)

### Stripe: Account × Environment

**Model**: Stripe Account (config) × Environment (data namespace)
- **Account**: Capabilities, webhooks, settings, pricing
- **Environment**: Live mode, test mode, sandboxes (up to 5 per account)
- **Relationship**: An account has multiple environments; customer:123 in live ≠ customer:123 in test

**Use cases**: Test/staging/production isolation, integration testing

**Our alignment**: WorkspaceID ≈ Application Environment, Universe ≈ (Stripe Account, Environment) pair

**Sources**:
- [Stripe testing environments](https://stripe.com/docs/testing)
- [Stripe sandboxes](https://stripe.com/docs/sandboxes)

### Vercel: Project × Environment

**Model**: Project (application) × Environment (runtime namespace)
- **Project**: Git repo, deployments, domains, config
- **Environment**: Local, Preview, Production, custom (staging, qa)
- **Relationship**: Each project has multiple environments with separate config/data

**Use cases**: Dev/preview/prod isolation, SaaS per-tenant data

**Our alignment**: WorkspaceID ≈ Project, Universe ≈ Environment

**Sources**:
- [Vercel environments](https://vercel.com/docs/concepts/environments)

### Common Pattern

All these systems use a **two-dimensional isolation model**:
- **Axis A**: Operational/config boundary (account, workspace, project)
- **Axis B**: Data/namespace boundary (database, catalog, dataset, environment)
- **Many-to-many relationship** via sharing mechanisms

**Our terminology is validated**: Workspace (operational) × Universe (data/namespace) aligns with industry patterns.

---

## Terminology Rationale

### Why "WorkspaceID" (not "TenantID", "RegionID", "AccountID")?

**Workspace** conveys:
- ✅ **Operational boundary** - a place where work happens
- ✅ **Configuration scope** - workspaces have their own settings
- ✅ **Team/organizational unit** - maps to how companies organize
- ✅ **Industry precedent** - Databricks, Slack, Figma, Notion all use "workspace"

**Rejected alternatives**:
- **TenantID**: Too generic; doesn't convey operational vs data distinction
- **RegionID**: Too narrow; workspaces aren't always regions (could be divisions, BUs)
- **AccountID**: Overloaded term (customer accounts, AWS accounts, Stripe accounts)
- **DeploymentID**: Too infrastructure-focused; misses organizational aspect

### Why "Universe" (not "Environment", "Namespace", "Realm")?

**Universe** conveys:
- ✅ **Reality/timeline semantics** - "which universe does this data live in?"
- ✅ **Identity scoping** - customer:123 in this universe ≠ customer:123 in that universe
- ✅ **Simulation/what-if** - parallel universes for scenarios
- ✅ **Memorable and evocative** - easy to explain: "production universe", "simulation universe"

**Rejected alternatives**:
- **Environment**: Overloaded (dev/test/prod, but also runtime environment, execution context)
- **Namespace**: Too generic/technical; doesn't convey identity scoping
- **Realm**: Domain-like but less intuitive than Universe
- **Timeline**: Good for temporal aspect, but doesn't convey namespace/identity scoping
- **Context**: Too generic; every system has "context"

**Why NOT "UniverseID"?**

Initially we considered `UniverseID` (parallel to `WorkspaceID`), but changed to just `Universe` because:
- WorkspaceID is likely a UUID (strong typing, generated)
- Universe is a **namespaced string** chosen by humans (`"production"`, `"test"`, `"sim-q4-2024"`)
- The `-ID` suffix implies system-generated identifier; Universe is user-defined
- Parallel: Stripe uses "live mode"/"test mode" (named states), not "environment ID"

---

## Implementation Details

### Value Objects

Both dimensions are implemented as DDD value objects:

```go
// WorkspaceID - represents operational boundary
type EventPayloadWorkspaceID struct {
    value string  // UUID or organization-assigned identifier
}

func NewEventPayloadWorkspaceID(value string) (EventPayloadWorkspaceID, error) {
    if value == "" {
        return EventPayloadWorkspaceID{}, fmt.Errorf("workspace ID is required")
    }
    return EventPayloadWorkspaceID{value: value}, nil
}

// Universe - represents data/namespace boundary
type EventPayloadUniverse struct {
    value string  // Human-readable: "production", "test", "sim-q4-2024"
}

func NewEventPayloadUniverse(value string) (EventPayloadUniverse, error) {
    if value == "" {
        return EventPayloadUniverse{}, fmt.Errorf("universe is required")
    }
    return EventPayloadUniverse{value: value}, nil
}
```

### Services Integration

**Ingestion Service**:
```go
// Schema lookup by workspace
func Ingest(eventPayload meters.EventPayload) (meters.Event, error) {
    // Look up schema: (WorkspaceID, EventType) → Schema
    schema := lookupSchema(eventPayload.WorkspaceID, eventPayload.EventType)

    // Type properties using workspace schema
    measures, dimensions := applySchema(eventPayload.Properties, schema)

    // Build Event preserving both dimensions
    spec := meters.EventSpec{
        TransactionID: eventPayload.TransactionID.ToString(),
        WorkspaceID:   eventPayload.WorkspaceID.ToString(),
        Universe:      eventPayload.Universe.ToString(),  // Preserved for downstream
        EventType:     eventPayload.EventType.ToString(),
        Subject:       eventPayload.Subject.ToString(),     // Scoped to Universe
        Timestamp:     eventPayload.Timestamp.ToTime(),
        Measures:      measures,
        Dimensions:    dimensions,
    }

    return meters.NewEvent(spec)
}
```

**Metering Service** (future):
```go
// Customer resolution scoped by Universe
func resolveCustomer(universe Universe, subject Subject) (CustomerID, error) {
    // (Universe, Subject) → CustomerID
    // Same subject in different universes = different customers
    return customerRegistry.Lookup(universe, subject)
}

// Contract binding
func bindContract(workspaceID WorkspaceID, universe Universe,
                   customerID CustomerID, eventType EventType) (ContractID, error) {
    // Contracts are scoped to (WorkspaceID, Universe, CustomerID)
    return contractService.FindActiveContract(workspaceID, universe, customerID, eventType)
}
```

---

## Migration Patterns

### Pattern 1: Adding a New Region

**Scenario**: Expanding from US to EU.

**Steps**:
1. Create new workspace: `acme-eu`
2. Define EU-specific event schemas for `acme-eu`
3. Point `acme-eu` workspace at `"production"` universe (shared customers)
4. Start ingesting EU events: `(acme-eu, "production", "customer:123", ...)`
5. Billing automatically aggregates across `acme-us` and `acme-eu` for customer:123

**Key point**: Customer identity (universe) stays the same; operational boundary (workspace) expands.

### Pattern 2: Creating a Test Environment

**Scenario**: Need isolated testing without affecting production.

**Steps**:
1. Use existing workspace: `acme-us`
2. Create new universe: `"test"`
3. Ingest test events: `(acme-us, "test", "customer:123", ...)`
4. `customer:123` in `"test"` is completely separate from `customer:123` in `"production"`

**Key point**: Operational boundary (workspace) stays the same; data namespace (universe) expands.

### Pattern 3: Post-Merger Integration

**Scenario**: Acme acquires WidgetCo.

**Phase 1 - Parallel Operation**:
```
Acme events:     (acme-us, "production", "customer:123", ...)
WidgetCo events: (widgetco, "production", "customer:123", ...)
→ Different workspaces, same universe name, but effectively isolated
```

**Phase 2 - Legacy Universes**:
```
Acme events:     (acme-us, "production", "customer:123", ...)
WidgetCo events: (widgetco, "legacy-widget", "customer:123", ...)
→ Explicit universe separation during migration
```

**Phase 3 - Unified Workspace** (optional):
```
Migrate WidgetCo schemas to acme-us workspace:
All events: (acme-us, "production", "customer:NEW_ID", ...)
→ Full integration complete
```

### Pattern 4: Simulation to Production Promotion

**Scenario**: Validated Q4 pricing in simulation, now promote to production.

**Approach A - Configuration Promotion**:
1. Test rules in `(acme-us, "simulation-q4", ...)`
2. Validate results
3. Apply same pricing rules to `(acme-us, "production", ...)`
4. Keep simulation universe for historical analysis

**Approach B - Data Migration** (less common):
1. Copy configuration from `"simulation-q4"` to `"production"`
2. Do NOT copy data (universes keep separate customer namespaces)

**Key point**: Promote configuration/rules, not data. Universes maintain separate customer identities.

---

## Design Principles

### 1. Explicit Over Implicit

Both dimensions are **first-class fields** in event structures:
- Not derived from other fields
- Not inferred from context
- Always present and validated

### 2. Orthogonality

Workspace and Universe are **completely independent**:
- Can change workspace without changing universe
- Can change universe without changing workspace
- No hidden coupling between dimensions

### 3. Many-to-Many by Design

The relationship is **intentionally many-to-many**:
- Not hierarchical (workspace doesn't "own" universe)
- Not one-to-many (workspace can have multiple universes)
- Explicit intersection: `(WorkspaceID, Universe)` is the unit of isolation

### 4. Human-Readable Universes

Universe values are **meaningful strings**:
- `"production"`, `"test"`, `"staging"` - standard environments
- `"simulation-q4-2024"`, `"sim-new-pricing"` - scenario names
- `"legacy-acme"`, `"legacy-widget"` - migration namespaces
- Not UUIDs or system-generated IDs

### 5. Customer Identity Scoping

Customer identity is **always scoped to universe**:
- `("production", "customer:123")` is one entity
- `("test", "customer:123")` is a different entity
- No "global" customer ID that spans universes

---

## Consequences

### Positive

1. **Solves multi-region with global customers**
   - Regional schemas + shared customer namespace
   - No customer duplication across regions

2. **Clean test/prod/simulation isolation**
   - Same customer IDs, different universes
   - No risk of test data affecting production

3. **Flexible merger integration**
   - Universe separation prevents ID collisions
   - Gradual migration without big-bang cutover

4. **Enables what-if scenarios**
   - Parallel universes for simulations
   - Risk-free experimentation

5. **Industry-validated pattern**
   - Snowflake, Databricks, BigQuery, AWS all use similar models
   - Strong precedent for many-to-many relationship

6. **Explicit and queryable**
   - Both dimensions are first-class fields
   - Easy to filter, aggregate, and analyze by workspace or universe

### Negative

1. **Increased complexity**
   - Two dimensions instead of one
   - Must think about both workspace and universe for every event

2. **Data model changes**
   - All existing services must handle both dimensions
   - Migration from single TenantID required

3. **Customer identity scoping**
   - Customer lookup now requires universe parameter
   - Can't ask "who is customer:123?" without universe context

4. **Learning curve**
   - Team must understand workspace vs universe distinction
   - New mental model to learn

### Mitigations

1. **Clear documentation** (this document, diagrams, examples)
2. **Explicit naming** (WorkspaceID vs Universe, not generic "Dimension1/Dimension2")
3. **Industry examples** (Snowflake, Databricks) to aid understanding
4. **Value objects** enforce presence and validation of both dimensions
5. **Use cases** demonstrate practical benefits

---

## Open Questions

1. **Universe lifecycle**: Who creates universes? Is there a Universe registry/catalog?
2. **Cross-universe queries**: Do we ever need to query across universes (e.g., compare production vs simulation)?
3. **Universe metadata**: Should universes have metadata (created_at, description, purpose)?
4. **Workspace metadata**: Should workspaces have metadata beyond ID (region, owner, purpose)?
5. **Default universe**: Is there a "default" universe if not specified, or always required?
6. **Universe naming conventions**: Should we enforce patterns like `{env}-{purpose}` or freeform?
7. **Historical data**: When a customer migrates universes, do we preserve history in old universe?

---

## Future Considerations

### Universe as a First-Class Entity

Currently Universe is a string field. We may want to introduce a Universe entity:

```go
type Universe struct {
    Name        string    // "production", "test", "sim-q4-2024"
    Description string    // Human-readable description
    Purpose     string    // "production", "testing", "simulation", "legacy"
    CreatedAt   time.Time
    CreatedBy   string
}
```

**Benefits**:
- Centralized universe registry
- Metadata for governance and auditing
- Validation of universe names
- Discovery of available universes

### Workspace as a First-Class Entity

Similarly, Workspace could become an entity:

```go
type Workspace struct {
    ID          string    // UUID or human-readable slug
    Name        string    // "Acme US", "Acme EU"
    Region      string    // "us-east-1", "eu-west-1"
    Owner       string    // Team or individual responsible
    EventSchemas map[EventType]Schema
}
```

### Cross-Universe Data Flows

Patterns for moving data between universes:
- **Promotion**: test → staging → production
- **Forking**: production → simulation (copy for what-if)
- **Migration**: legacy-system-a → production (merger)
- **Archival**: production → archive (retention)

### Multi-Workspace Queries

Aggregations across workspaces within a universe:
- Total usage for customer:123 across all regions (workspaces) in "production" universe
- Global billing for customer with events from acme-us, acme-eu, acme-apac

### Universe Hierarchies

Should universes support hierarchy?
```
production
├── production-stable
└── production-canary

test
├── test-integration
└── test-e2e
```

Probably not needed initially, but consider for future.

---

## References

### Internal Documentation
- `ubiquitous-language.md` - Core domain terminology
- `order-to-value.md` - Process architecture
- Commit f73f8b4 - Workspace and Universe refactor implementation

### Research & Validation
- `chats/workspaces/01-initial-audit.md` - Industry research on two-dimensional isolation patterns

### Industry Examples
- [Snowflake Secure Data Sharing](https://docs.snowflake.com/en/user-guide/data-sharing-intro) - Account × Database
- [Databricks Unity Catalog](https://docs.databricks.com/data-governance/unity-catalog/) - Workspace × Metastore
- [BigQuery Multi-Tenancy](https://cloud.google.com/bigquery/docs/best-practices-multi-tenancy) - Project × Dataset
- [AWS Lake Formation](https://aws.amazon.com/blogs/big-data/design-patterns-for-an-enterprise-data-lake-using-aws-lake-formation-cross-account-access/) - Account × Database
- [Stripe Environments](https://stripe.com/docs/testing) - Account × Environment
- [Vercel Environments](https://vercel.com/docs/concepts/environments) - Project × Environment

---

## Conclusion

The Workspace × Universe model provides a robust, industry-validated approach to two-dimensional isolation that solves our core challenges:

✅ Multi-region with global customers
✅ Test/prod/simulation isolation
✅ Post-merger integration
✅ What-if scenarios
✅ Flexible operational boundaries

By making both dimensions **explicit, orthogonal, and many-to-many**, we gain flexibility that a single-dimension model cannot provide, while following patterns proven at scale by Snowflake, Databricks, BigQuery, and AWS.
