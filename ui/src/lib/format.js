const dateFormatter = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
const dateTimeFormatter = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
const relativeFormatter = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })

const toDate = (value) => {
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

export const formatDate = (value, fallback = '—') => {
  const date = toDate(value)
  return date ? dateFormatter.format(date) : fallback
}

export const formatDateTime = (value, fallback = '—') => {
  const date = toDate(value)
  return date ? dateTimeFormatter.format(date) : fallback
}

const relativeUnits = [
  ['year', 365 * 24 * 3600],
  ['month', 30 * 24 * 3600],
  ['week', 7 * 24 * 3600],
  ['day', 24 * 3600],
  ['hour', 3600],
  ['minute', 60],
]

export const formatRelative = (value, fallback = 'Never') => {
  const date = toDate(value)
  if (!date) return fallback
  const seconds = (date.getTime() - Date.now()) / 1000
  for (const [unit, size] of relativeUnits) {
    if (Math.abs(seconds) >= size) {
      return relativeFormatter.format(Math.round(seconds / size), unit)
    }
  }
  return 'just now'
}

export const formatLifetime = (hours) => {
  const value = Number(hours || 0)
  if (value <= 0) return 'not set'
  if (value % 24 === 0) return value === 24 ? '1 day' : `${value / 24} days`
  return value === 1 ? '1 hour' : `${value} hours`
}

export const formatExpiration = (slug) => {
  if (!slug) return ''
  if (slug.expiration_type === 'duration') {
    return slug.expiration_days === 1 ? 'Expires after 1 day' : `Expires after ${slug.expiration_days} days`
  }
  if (slug.expiration_type === 'fixed_date') return `Expires ${formatDate(slug.fixed_expires_at)}`
  return 'Never expires'
}

export const formatSeats = (count) => (Number(count) === 1 ? '1 seat' : `${count} seats`)

export const parseJSONObject = (raw, label = 'Value') => {
  const trimmed = (raw || '').trim()
  if (!trimmed) return {}

  let parsed
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    throw new Error(`${label} is not valid JSON`)
  }
  if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error(`${label} must be a JSON object`)
  }
  return parsed
}

export const prettyJSON = (value) => JSON.stringify(value || {}, null, 2)

export const hasKeys = (value) => Boolean(value) && typeof value === 'object' && Object.keys(value).length > 0

export const errorMessage = (err, fallback) => err?.response?.data?.error || err?.message || fallback

// Expiry dates are picked as a calendar day and expire at the end of that day, local time.
export const toDateInputValue = (value) => {
  const date = toDate(value)
  if (!date) return ''
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 10)
}

export const endOfLocalDay = (value) => {
  const match = (value || '').match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!match) return null
  const [, year, month, day] = match.map(Number)
  const date = new Date(year, month - 1, day, 23, 59, 59)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) return null
  return date
}

const reasonLabels = {
  not_found: 'License not found',
  revoked: 'License is revoked',
  expired: 'License has expired',
  activation_limit_reached: 'All seats are in use',
  fingerprint_not_active: 'This device is not activated',
}

export const reasonLabel = (reason) => reasonLabels[reason] || reason || 'Not valid'

// Reads the exp claim of a JWT for display only. Nothing here verifies the signature.
export const jwtExpiry = (token) => {
  try {
    const payload = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
    const claims = JSON.parse(atob(payload.padEnd(payload.length + ((4 - (payload.length % 4)) % 4), '=')))
    return claims.exp ? new Date(claims.exp * 1000) : null
  } catch {
    return null
  }
}
