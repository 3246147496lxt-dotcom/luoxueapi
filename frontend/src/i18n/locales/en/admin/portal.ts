import overview from './overview'
import resources from './resources'

// User-facing pages still share a small set of mature usage/group labels with
// the admin console. Keep that compatibility surface explicit so the user
// entry does not load the complete admin locale tree.
export default {
  dashboard: overview.dashboard,
  users: overview.users,
  groups: overview.groups,
  usage: resources.usage,
  redeem: resources.redeem,
}
