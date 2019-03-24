/**
 * @typedef User
 * @property {bigint=} id
 * @property {string} username
 * @property {string=} avatarURL
 */

/**
 * @typedef UserProfile
 * @property {bigint=} id
 * @property {string=} email
 * @property {string} username
 * @property {string=} avatarURL
 * @property {number} followersCount
 * @property {number} followeesCount
 * @property {boolean} me
 * @property {boolean} following
 * @property {boolean} followeed
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
 * @typedef Comment
 * @property {bigint} id
 * @property {string} content
 * @property {number} likesCount
 * @property {string|Date} createdAt
 * @property {User=} user
 * @property {boolean} mine
 * @property {boolean} liked
 */

/**
 * @typedef CreatePostInput
 * @property {string} content
 * @property {boolean=} NSFW
 * @property {string=} spoilerOf
 */

/**
 * @typedef ToggleFollowOutput
 * @property {number} followersCount
 * @property {boolean} following
 */

/**
 * @typedef ToggleLikeOutput
 * @property {number} likesCount
 * @property {boolean} liked
 */

export default undefined
