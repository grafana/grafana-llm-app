# Change Log

All notable changes to this project will be documented in this file.

## v1.7.1

- Add vector search support for LLM integration

## v1.7.0

- Add experimental LLM support

## v1.6.2

- #Feature: Allow customizing the list of built-in authentication methods

## v1.6.1

- Fix type changes in EditorList

## v1.6.0

- Add new `ConnectionSettings` and `AdvancedHttpSettings` components to simplify migration from `DataSourceHttpSettings` component.
- Improve docs for some components.

## v1.5.1

- Fix Auth component to prevent it from failing when it is used in Grafana 8

## v1.5.0

- Introduce treeshaking by rewriting rollup build configs to include both cjs and esm builds

## v1.4.3

- `Auth` and `DataSourceDescription` components: change asterisk color (for marking required fields) from red to default

## v1.4.2

- Update `GenericConfigSection` component type for prop `description` to `ReactNode`

## v1.4.1

- Fix types for `Auth` component - allow any `jsonData`

## v1.4.0

- `DataSourceDescription` config editor component: added possibility to pass `className` + minor styling changes

## v1.3.0

- Add Auth component

## v1.2.0

- Add new ConfigSection, ConfigSubSection and DataSourceDescription components

## v1.1.0

- EditorList now accepts a ref to the Button for adding items

## v1.0.2

- Make EditorField tooltip selectable via keyboard

## v1.0.1

- Specify Grafana packages as dev- and peer dependencies

## v1.0.0

- Add back QueryEditor components

## v0.0.1

- Initial Release
