import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import admin from './admin'
import misc from './misc'
import chat from './chat'
import personalSettings from './personalSettings'
import desktop from './desktop'
import quotaViewerLanding from './quotaViewerLanding'
import skills from './skills'
import library from './library'

export default {
  ...landing,
  ...common,
  ...dashboard,
  admin,
  ...misc,
  ...chat,
  ...personalSettings,
  ...desktop,
  ...quotaViewerLanding,
  ...skills,
  ...library,
}
