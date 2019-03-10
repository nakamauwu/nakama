/**
 * @typedef User
 * @property {bigint=} id
 * @property {string} username
 * @property {string=} avatarURL
 */

/**
 * @typedef DevLoginOutput
 * @property {string} token
 * @property {string|Date} expiresAt
 * @property {User} authUser
 */

/**
 * @typedef Post
 * @property {bigint} id
 * @property {string} content
 * @property {boolean} NSFW
 * @property {string=} spoilerOf
 * @property {number} likesCount
 * @property {number} commentsCount
 * @property {string|Date} createdAt
 * @property {User=} user
 * @property {boolean} mine
 * @property {boolean} liked
 * @property {boolean} subscribed
 */

/**
 * @typedef TimelineItem
 * @property {bigint} id
 * @property {Post=} post
 */

/**
 * @typedef CreatePostInput
 * @property {string} content
 * @property {boolean=} NSFW
 * @property {string=} spoilerOf
 */

export default undefined
