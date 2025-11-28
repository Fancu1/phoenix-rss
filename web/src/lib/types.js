// @ts-check

/**
 * @typedef {Object} User
 * @property {number} id
 * @property {string} username
 */

/**
 * @typedef {Object} Feed
 * @property {number} id
 * @property {string} title
 * @property {string} url
 * @property {string} description
 * @property {string} created_at
 * @property {string} updated_at
 * @property {string|null} [custom_title] - User-defined custom title for this feed
 */

/**
 * @typedef {Object} Article
 * @property {number} id
 * @property {number} feed_id
 * @property {string} title
 * @property {string} url
 * @property {string} description
 * @property {string} content
 * @property {string} created_at
 * @property {string} updated_at
 * @property {boolean} read
 * @property {boolean} starred
 * @property {string} published_at
 * @property {string|null} summary
 * @property {string|null} processing_model
 * @property {string|null} processed_at
 * @property {string} [feed_name]
 */

/**
 * @typedef {Object} Toast
 * @property {string} id
 * @property {'success'|'error'|'warning'|'info'} type
 * @property {string} message
 */

/**
 * @typedef {Object} AuthState
 * @property {string|null} token
 * @property {User|null} user
 * @property {'unknown'|'authenticated'|'anonymous'} status
 */

/**
 * @typedef {Object} UIState
 * @property {boolean} sidebarOpen
 * @property {'light'|'dark'} theme
 * @property {Toast[]} toasts
 */

export { };

