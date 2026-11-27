import { reactive } from 'vue'

// App-wide UI state shared between the header, the issue-license drawer and the views.
export const ui = reactive({
  issueDrawerOpen: false,
  // Bumped whenever licenses change outside the Licenses view, so it can refresh.
  licensesVersion: 0,
  toasts: [],
})

export const openIssueDrawer = () => {
  ui.issueDrawerOpen = true
}

export const closeIssueDrawer = () => {
  ui.issueDrawerOpen = false
}

export const notifyLicensesChanged = () => {
  ui.licensesVersion += 1
}

let toastId = 0

export const toast = (message, tone = 'success') => {
  const id = ++toastId
  ui.toasts.push({ id, message, tone })
  window.setTimeout(() => dismissToast(id), tone === 'error' ? 6000 : 3200)
}

export const toastError = (message) => toast(message, 'error')

export const dismissToast = (id) => {
  const index = ui.toasts.findIndex((item) => item.id === id)
  if (index !== -1) ui.toasts.splice(index, 1)
}
