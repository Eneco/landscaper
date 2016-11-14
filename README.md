# Rough outline of the flow
- Prepare release configurations
    - Read secrets from environment
    - Create Release structs
- Diff
    - Get current releases and their configurations/secrets from k8s namespace
    - Compare Releases with current desired state
- Apply
    - Create secrets in k8s namespace for all releases
    - Run each ReleaseTask

# Secrets design principles
- Charts should use k8s secrets to get values into the environment of containers
- Secrets should be a separate array in the component file

# Rough brain dump of required entities
## Configuration
- Read()

## Secrets
- Read()

## Release
- Chart
- Version

## Component
- Name
- Release
- Configuration
- Secrets
- Equals(Component) bool

## Executor
- Apply(desired []Component, current []Component)
- CreateComponent(Component)
- UpdateComponent(Component)
- DeleteComponent(Component)

## ComponentProvider
- Current() []Component
- Desired() []Component

## SecretProvider
- Read() []Secret
