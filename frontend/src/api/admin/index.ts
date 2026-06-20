import dashboardAPI from './dashboard'
import groupsAPI from './groups'
import opsAPI from './ops'
import settingsAPI from './settings'
import systemAPI from './system'

export const adminAPI = {
  dashboard: dashboardAPI,
  groups: groupsAPI,
  ops: opsAPI,
  settings: settingsAPI,
  system: systemAPI
}

export {
  dashboardAPI,
  groupsAPI,
  opsAPI,
  settingsAPI,
  systemAPI
}

export default adminAPI
