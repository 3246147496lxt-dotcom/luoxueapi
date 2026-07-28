import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import audit from './audit'
import modelCatalog from './modelCatalog'
import documentation from './documentation'
import desktopDiagnostics from './desktopDiagnostics'

export default {
  ...overview,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
  ...audit,
  ...modelCatalog,
  ...documentation,
  ...desktopDiagnostics,
}
