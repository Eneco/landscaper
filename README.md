h2. Rough outline of the flow

# Prepare release configurations
## Read secrets from environment
## Create Release structs
# Diff
## Get current releases and their secrets from k8s namespace
## Compare Releases with current desired state
# Apply
## Create secrets in k8s namespace for all releases
## Run each ReleaseTask

h2. Rough brain dump of required entities

*Action* (CREATE/UPDATE/DELETE)

*Configuration*
* Read()
* Compare(Configuration) (Action) 


*Secrets*
* Read()
* Compare(Secrets) (Action)


*Release*
* Name
* Chart
* Version
* Configuration
* Secrets
* Compare(Release) (Diff)


*Diff*
* Release
* Action


*Executor*
* Apply([]Diff)


*ReleaseProvider*
* Current()
* Create(Release)
* Update(Release)
* Delete(Release)


*ConfigurationProvider*
* Read()

	
*SecretProvider*
* Read()
