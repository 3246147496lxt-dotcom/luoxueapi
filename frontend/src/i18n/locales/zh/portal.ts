import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import misc from './misc'
import chat from './chat'
import personalSettings from './personalSettings'
import desktop from './desktop'
import quotaViewerLanding from './quotaViewerLanding'
import skills from './skills'
import admin from './admin/portal'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...misc,
  ...chat,
  ...personalSettings,
  ...desktop,
  ...quotaViewerLanding,
  ...skills,
  admin,
}
