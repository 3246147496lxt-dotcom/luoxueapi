import overview from './overview'
import resources from './resources'

// 用户端仍与后台复用少量成熟的用量、分组文案。显式限定兼容面，避免
// 用户入口重新加载完整后台语言包。
export default {
  dashboard: overview.dashboard,
  users: overview.users,
  groups: overview.groups,
  usage: resources.usage,
  redeem: resources.redeem,
}
