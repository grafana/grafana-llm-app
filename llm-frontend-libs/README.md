# Grafana Experimental

The main difference between grafana/experimental and grafana/ui is how the components are versioned. Having a separate "experimental" package allows us to release breaking changes when necessary while the grafana/ui package follows a slower-moving policy.

As developers use and test the components and report issues, the maintainers learn more about shortcomings of the components. The older and more used a component is, the less likely it is that new issues will be found and subsequently need to introduce breaking changes.

For a component to be ready to move to the grafana/ui package, the following criteria are considered:

- It needs to match the code quality of the grafana/ui components. It doesn't have to be perfect to be part of grafana/ui, but the component should be reliable enough that developers can depend on it.
- Each component needs stories.
- Each component needs type definitions.
- Requires good test coverage. Some of the grafana/experimental components don't currently have comprehensive tests.
- It needs to have a low probability of a breaking change in the short/medium future. For instance, if it needs a new feature that will likely require a breaking change, it may be preferable to delay it being added to grafana/ui.

# CONTRIBUTING

As we are planning to get this repository to a more stable state please please write tests for your components.

If you want to use your local development version in another repo run `yarn add link:"/yar/path/to/grafana-experimental"`
