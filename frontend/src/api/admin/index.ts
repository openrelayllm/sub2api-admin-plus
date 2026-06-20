import dashboardAPI from './dashboard'
import groupsAPI from './groups'
import adminPlusAPI from './adminPlus'
import opsAPI from './ops'
import settingsAPI from './settings'
import systemAPI from './system'

export const adminAPI = {
  dashboard: dashboardAPI,
  groups: groupsAPI,
  adminPlus: adminPlusAPI,
  ops: opsAPI,
  settings: settingsAPI,
  system: systemAPI
}

export {
  dashboardAPI,
  groupsAPI,
  adminPlusAPI,
  opsAPI,
  settingsAPI,
  systemAPI
}

export default adminAPI
