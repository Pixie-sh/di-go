# di-go

A powerful dependency injection library for Go that supports configuration management, JSON templating, and flexible service registration.

## Features

- **Type-safe Dependency Injection**: Leverage Go generics for compile-time type safety
- **Configuration Management**: Built-in support for configuration injection and resolution
- **JSON Templating**: Advanced JSON configuration with variable interpolation and shared sections
- **Singleton Support**: Automatic singleton management for services
- **Flexible Service Registration**: Support for both factory functions and configuration-based registration

## Quick Start

### Basic Service Registration
```go
// Register a service with a factory function 
Register[MyService](func(c Context, opts *RegistryOpts) (MyService, error) { return MyService{}, nil })

// Create an instance 
ctx := NewContext(config) service, err := Create[MyService](ctx)
```

 
### Configuration-Based Services

```go
// Register configuration type 
RegisterConfiguration[RedisConfig](ConfigurationLookup[RedisConfig])

// Create configuration instance 
redisConfig, err := CreateConfiguration[RedisConfig](ctx, WithConfigNode("redis"))
```

### Advanced JSON Configuration

```json
{
  "shared": {
    "database": {
      "host": "localhost",
      "port": 5432
    }
  },
  "services": {
    "user_service": {
      "db": "{di.shared.database}"
    },
    "order_service": {
      "db": "{di.shared.database}"
    }
  }
}
``` 

## Core Concepts

### Context
The `Context` is the central container that holds all registered services and configurations.

### Registration Options
- `WithToken(token)`: Register service with a specific identifier
- `WithConfigNode(node)`: Specify configuration node for service creation
- `WithOpts(opts)`: Pass additional registry options

### Configuration Resolution
The library supports automatic resolution of JSON templates with:
- Shared configuration sections (`$shared`)
- Variable interpolation (`${di.path.to.value}`)
- Nested object references

## API Reference

### Core Functions

- `Register[T](factory, ...opts)`: Register a service factory
- `RegisterConfiguration[T](lookup)`: Register a configuration type
- `Create[T](context, ...opts)`: Create service instance
- `CreateConfiguration[T](context, ...opts)`: Create configuration instance
- `NewContext(config)`: Create new DI context
- `UnmarshalJSONWithDIResolution(data, target)`: Parse JSON with template resolution

### Configuration Interface

```go
type Configuration interface {
    LookupNode(lookupPath string) (any, error)
}
```


## High Level architecture of di.Registry

```mermaid
graph TB
subgraph "di-go Architecture"
subgraph "Core Registry System"
R[Registry Interface] --> DR[diRegistry Implementation]
DR --> REG[registrations map]
DR --> CREG[configurationRegistrations map]
DR --> HOT[hotInstances map]
end

subgraph "Registration Flow"
RF1[Register&lt;T&gt;] --> RST[registerSingleWithToken]
RF2[RegisterPair&lt;T,CT&gt;] --> RPT[registerPairWithToken]
RF3[RegisterConfiguration&lt;T&gt;] --> RSCT[registerSingleConfigurationWithToken]

RST --> REG
RPT --> REG
RPT --> CREG
RSCT --> CREG
end

subgraph "Creation Flow"
CF1[Create&lt;T&gt;] --> CST[createSingleWithToken]
CF2[CreatePair&lt;T,CT&gt;] --> CPT[createPairWithToken]
CF3[CreateConfiguration&lt;T&gt;] --> CSCT[createSingleConfigurationWithToken]

CST --> CHK1{Check Hot Instance}
CPT --> CHK2{Check Hot Instance}
CSCT --> CHK3{Check Hot Instance}

CHK1 -->|Found| RET1[Return Cached]
CHK1 -->|Not Found| EXEC1[Execute Factory]
CHK2 -->|Found| RET2[Return Cached]
CHK2 -->|Not Found| EXEC2[Execute Factory]
CHK3 -->|Found| RET3[Return Cached]
CHK3 -->|Not Found| EXEC3[Execute Factory]

EXEC1 --> STORE1[Store in Hot Instances]
EXEC2 --> STORE2[Store in Hot Instances]
EXEC3 --> STORE3[Store in Hot Instances]
end

subgraph "Type System"
IT[InjectionToken] --> TN[TypeName Generation]
TN --> PTN[PairTypeName for Dependencies]
PTN --> REG
PTN --> CREG

TA[safeTypeAssert] --> TC[Type Conversion]
TC --> RET1
TC --> RET2
TC --> RET3
end

subgraph "Registry Options"
RO[RegistryOpts] --> WO[WithOpts]
RO --> WR[WithRegistry]
RO --> WT[WithToken]
RO --> WCN[WithConfigNode]

WO --> RF1
WR --> RF1
WT --> RF1
WCN --> RF1
end

subgraph "Hot Memory Management"
HM[Hot Instances] --> GHI[GetHotInstance]
HM --> SHI[SetHotInstance]

GHI --> CHK1
GHI --> CHK2
GHI --> CHK3

SHI --> STORE1
SHI --> STORE2
SHI --> STORE3
end

subgraph "Handler Types"
CIH[CreateInstanceHandler]
CCH[CreateConfigurationHandler]
TCIH[TypedCreateInstanceHandler&lt;T,CT&gt;]
TCNCH[TypedCreateInstanceNoConfigHandler&lt;T&gt;]

TCIH --> CIH
TCNCH --> CCH
end
end

style DR fill:#e1f5fe
style REG fill:#f3e5f5
style CREG fill:#e8f5e8
style HOT fill:#fff3e0
style IT fill:#fce4ec
```

This README and diagram provide a comprehensive overview of the di-go library's architecture and capabilities. The library appears to be designed for complex enterprise applications where configuration management and dependency injection are critical concerns.

## Examples
See the test files for comprehensive examples including:
- Complex service hierarchies
- Configuration sharing and templating
- Singleton pattern implementation
- Multi-level dependency injection
