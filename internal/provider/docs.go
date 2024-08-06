package provider

const (
	mdDocProviderOverview = `
Yet another Terraform provider for PostgreSQL. This one is built using the latest and suggested practices for
Terraform providers. That means it is built using the (Terraform Plugin Framework)[https://developer.hashicorp.com/terraform/plugin/framework].

## ❗ READ BEFORE USE

* This provider is still in development and has a limited support for PostgreSQL resources.
* Check the [🏁 Roadmap](#-roadmap) for the list of supported resources.

## 🏁 Roadmap

| Name          | Resource | Data Source |
|---------------|:--------:|:-----------:|
| Event Trigger |    ✅    |     ✅      |
| Functions     |    🔜    |     🔜      |
| Database      |    🔜    |     🔜      |
| Schema        |    🔜    |     🔜      |
| Role          |    🔜    |     🔜      |

`
	mdDocResourceEventTrigger = `
Event Trigger is a PostgreSQL object that allows you to define a set of actions that should be executed when a certain event occurs.
They are are global objects for a particular database and are capable of capturing events from multiple tables.
(PostgreSQL Event Triggers)[https://www.postgresql.org/docs/current/event-triggers.html]`
)
