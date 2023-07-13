---
id: api-overview
title: API Overview
---

# API Overview

## API Discovery

API discovery routes are endpoints that provide information about the available API versions and resources provided by a Kubernetes API server.

1. `/` provides information about the available API versions in the cluster. It returns a list of supported API versions and their corresponding endpoints.
1. `/apis` provides a list of all available API groups and their versions. Each API group represents a set of related resources and functionalities.
1. `/apis/{group}` lists the versions available for a specific API group. For example, `/apis/kinds.grafana.com` provides the versions of the API group, which includes resources like GrafanaKind.
1. `/apis/{group}/{version}` provides a list of resources available in a specific API version of an API group. For example, `/apis/kinds.grafana.com/v1` lists the resources available in the `kinds.grafana.com` API group's `v1` version.

Discoverable API resources can be listed using `kubectl api-resources`:

```
$ kubectl api-resources                                           
NAME                         SHORTNAMES   APIVERSION                        NAMESPACED   KIND
accesspolicies                            core.kinds.grafana.com/v0-alpha   true         AccessPolicy
dashboards                                core.kinds.grafana.com/v0-alpha   true         Dashboard
folders                                   core.kinds.grafana.com/v0-alpha   true         Folder
librarypanels                             core.kinds.grafana.com/v0-alpha   true         LibraryPanel
playlists                                 core.kinds.grafana.com/v0-alpha   true         Playlist
preferences                               core.kinds.grafana.com/v0-alpha   true         Preferences
publicdashboards                          core.kinds.grafana.com/v0-alpha   true         PublicDashboard
rolebindings                              core.kinds.grafana.com/v0-alpha   true         RoleBinding
roles                                     core.kinds.grafana.com/v0-alpha   true         Role
teams                                     core.kinds.grafana.com/v0-alpha   true         Team
grafanaresourcedefinitions   grd,grds     kinds.grafana.com/v1              false        GrafanaResourceDefinition
```

It's also possible to view the raw discovery responses from the API server using `kubectl get --raw` with the endpoints listed above.

```
$ kubectl get --raw='/apis/core.kinds.grafana.com' | jq
{
  "kind": "APIGroup",
  "apiVersion": "v1",
  "name": "core.kinds.grafana.com",
  "versions": [
    {
      "groupVersion": "core.kinds.grafana.com/v0-alpha",
      "version": "v0-alpha"
    }
  ],
  "preferredVersion": {
    "groupVersion": "core.kinds.grafana.com/v0-alpha",
    "version": "v0-alpha"
  }
}
```
