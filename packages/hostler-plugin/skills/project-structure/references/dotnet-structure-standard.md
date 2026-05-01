# .NET Project Structure Standard

## When to Apply

If the project root contains a `*.sln` or `*.slnx` file, treat it as a .NET project and apply this standard.

## The 10 Rules

### 1. Bounded Context = 1:1 with Project
Each BC corresponds to exactly one .NET library project.
Project name: `{Namespace}.{BC}` (e.g. MyApp.Orders, MyApp.Payments)

### 2. Deploy Unit = {BC}.Host
Each deployment process is a `{BC}.Host` project (ASP.NET Core Web / Console / Worker).
AssemblyName: matches the deploy unit name.

### 3. Three Folders Inside the BC Library
```
{BC}/
├── Domain/           # Entities, value objects, Aggregates, domain events
├── Application/      # Use cases, services, DTOs, interfaces
├── Infrastructure/   # Technical implementations (DB, external APIs, messaging)
└── {BC}.csproj
```

### 4. Inside the Host Project
```
{BC}.Host/
├── Api/              # Minimal API endpoints (Controllers/ is also fine)
├── Health/           # IHealthCheck implementations
├── Middleware/       # HTTP middleware (optional)
├── ServiceRegistration.cs
├── Program.cs
└── {BC}.Host.csproj
```

### 5. namespace = Folder Path
The `namespace` declaration must match the file's physical folder path exactly.
Example: `src/MyApp.Orders/Domain/OrderAggregate.cs` → `namespace MyApp.Orders.Domain;`

### 6. Frontend Separation
Do not place a Node.js / React / Angular / SvelteKit project inside the .NET project folder (`src/`).
Separate them into a top-level `frontend/` folder and integrate via the Host Dockerfile multi-stage build.

### 7. SharedKernel vs Infrastructure
- SharedKernel: pure DDD building blocks (Entity, ValueObject, AggregateRoot, DomainEvent)
- Infrastructure: technical infra (logging, auth, health checks, Circuit Breaker, Rate Limiter)
Keep them as separate projects to preserve the purity of SharedKernel.

### 8. Project Reference Direction
```
SharedKernel ← Contracts ← {BC} ← {BC}.Host
                              ↑
                        Infrastructure
```
No direct BC-to-BC references. Integration only through Contracts (integration events).

### 9. Subdomain = Subfolder of Domain/
Express subdomains within a BC as subfolders of Domain/.
```
Domain/
├── OrderManagement/   # subdomain 1
├── Shipping/          # subdomain 2
└── Events/            # domain events
```

### 10. Solution Folders
```xml
<Solution>
  <Folder Name="/src/">...</Folder>
  <Folder Name="/tests/">...</Folder>
</Solution>
```

## Generalization Basis

These 10 rules are validated in real .NET projects and reflect common .NET patterns:
- Jason Taylor Clean Architecture Template
- Microsoft .NET Architecture Guide
- .NET Coding Conventions (namespace = folder)
- ASP.NET Core Hosting Patterns
- DDD Modular Monolith Standards
