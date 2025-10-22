# AI Rules for terraform-provider-postgresql

A modern and efficient implementation of a Terraform Provider for PostgreSQL, it applies the latest best practices for
Terraform providers using the Terraform Plugin Framework.

## PROJECT_STRUCTURE

Any directory or file that is not part of the project structure could be very important as well, just for the sake of
clarity this is a shortlist.

```tree
├── assets
├── docs
│	 ├── data-sources
│	 └── resources
├── examples
├── internal
│	 ├── client
│	 ├── helpers
│	 ├── pgclient
│	 │	 └── pgcustomtypes
│	 ├── provider
│	 │	 └── validators
│	 └── test1
├── templates
│	 ├── data-sources
│	 └── resources
└── tools
```

Breakdown of the project structure:

*`internal/client`: Contains deprecated code with the previous implementation of the PostgreSQL client.
*`internal/helpers`: Contains generic helper functions.
*`internal/pgclient`: Contains the current PostgreSQL client implementation.
*`internal/provider`: Contains the provider implementation (Provider setup, Resources, DataSources, Functions, etc).
*`internal/test`: Contains some important helpers for test purposes.
*`templates`: Contains the Markdown templates used by the Terraform Docs generation tool.
*`tools`: Contains the tools used to generate the documentation and other files.

## OVERALL_PRINCIPLES

1. **Review the code**: Check for correctness, efficiency, and readability.
2. **Check for best practices**: Ensure that the code follows the best practices for the language and framework used.
3. **Avoid unnecessary changes**: Only suggest changes that improve the code or fix issues. Avoid making changes that
   do not add value.
4. **Be concise**: Provide clear and concise feedback. Avoid long explanations unless necessary.
5. **Use examples**: If you suggest a change, provide an example of how to implement it.
6. **Be straightforward**: Focus on practical solutions that can be implemented easily. Avoid overly complex solutions
   unless necessary. Do not argue for the sake of arguing, but rather provide constructive feedback. Avoid any
   confirmation bias or unnecessary praise.
7. **Be thorough**: When either you or I suggest something is not working, investigate the issue thoroughly. Choose
   facts over opinions or feelings, don't be apologetic, and focus on finding the root cause of the issue and
   therefore the best solution.


## CODING_PRACTICES

### Guidelines for AI Support

#### SUPPORT_EXPERT

- Favor elegant, maintainable solutions over verbose code. Assume understanding of language idioms and design patterns.
- Highlight potential performance implications and optimization opportunities in suggested code.
- Frame solutions within broader architectural contexts and suggest design alternatives when appropriate.
- Focus comments on 'why' not 'what' - assume code readability through well-named functions and variables.
- Proactively address edge cases, race conditions, and security considerations without being prompted.
- When debugging, provide targeted diagnostic approaches rather than shotgun solutions.
- Suggest comprehensive testing strategies rather than just example tests, including considerations for mocking, test
  organization, and coverage.

### Guidelines for DOCUMENTATION

#### DOC_UPDATES

- Make sure the code is well-documented and avoid overly verbose documentation.
- Keep Project level documentation updated in the README.md
- Follow the expected terraform-provider documentation libraries and templates. Keep such documentation in sync with
  any new, deleted, or updated features.

### Guidelines for ARCHITECTURE

#### DDD

- Define bounded contexts to separate different parts of the domain with clear boundaries
- Implement ubiquitous language within each context to align code with business terminology
- Create rich domain models with behavior, not just data structures, for PostgreSQL objects and entities.
- Use value objects for concepts with no identity but defined by their attributes
- Implement domain events to communicate between bounded contexts
- Use aggregates to enforce consistency boundaries and transactional integrity

#### CLEAN_ARCHITECTURE

- Strictly separate code into layers: entities, use cases, interfaces, and frameworks
- Ensure dependencies point inward, with inner layers having no knowledge of outer layers
- Implement domain entities that encapsulate {{business_rules}} without framework dependencies
- Use interfaces (ports) and implementations (adapters) to isolate external dependencies
- Create use cases that orchestrate entity interactions for specific business operations
- Implement mappers to transform data between layers to maintain separation of concerns

### Guidelines for STATIC_ANALYSIS

#### SONARQUBE

- Configure quality gates with appropriate thresholds for {{critical_metrics}}
- Set up branch analysis to track quality metrics across different development branches
- Implement custom quality profiles tailored to the Go language and project standards
- Use SonarLint IDE integration to catch issues before they reach the repository
- Configure security hotspot reviews as part of the development workflow
- Use SonarLint IDE integration to catch issues before they reach the repository

#### CODECOV

- Set minimum coverage thresholds for {{critical_code_paths}} to ensure adequate testing
- Configure path-specific coverage targets based on risk assessment
- Use coverage flags to categorize tests (unit, integration, e2e) for better reporting
- Implement coverage checks in CI/CD pipelines to prevent coverage regression
- Configure branch coverage in addition to line coverage for more thorough analysis
- Set up pull request comments to highlight coverage changes during review

## TESTING

### Guidelines for UNIT

#### Testify

- Use [Testify]{github.com/stretchr/testify} with Go for testing assertions, etc.
- Leverage mock functions and spies for isolating units of code
- Leverage expect assertions with specific matchers
- Implement code coverage reporting with meaningful targets
- Leverage fake timers for testing time-dependent functionality

### Guidelines for INTEGRATION

#### terraform-plugin-testing and testcontainers-go

- Avoid flaky tests by ensuring proper setup and teardown of test environments, especially using Testcontainers.
- Use terraform-plugin-testing for setting up the Acceptance tests for Terraform providers, which involves resources,
  datasources, and functions.
- Cover only the most critical cases during the Acceptance tests. 

