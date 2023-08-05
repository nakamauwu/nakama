/**
 * @typedef User
 * @prop {string=} id
 * @prop {string} username
 * @prop {string=} avatarURL
 */

/**
 * @typedef UserProfile
 * @prop {string=} id
 * @prop {string=} email
 * @prop {string} username
 * @prop {string=} avatarURL
 * @prop {number} followersCount
 * @prop {number} followeesCount
 * @prop {boolean} me
 * @prop {boolean} following
 * @prop {boolean} followeed
 */

/**
 * @typedef DevLoginOutput
 * @prop {string} token
 * @prop {string|Date} expiresAt
 * @prop {User} user
 */

/**
 * @typedef {Post & TimelineItemExt} TimelineItem
 */


/**
 * @typedef Post
 * @prop {string} id
 * @prop {string} content
 * @prop {boolean} nsfw
 * @prop {string=} spoilerOf
 * @prop {ReactionCount[]} reactions
 * @prop {number} commentsCount
 * @prop {string[]} mediaURLs
 * @prop {string|Date} createdAt
 * @prop {string|Date} updatedAt
 * @prop {User=} user
 * @prop {boolean} mine
 * @prop {boolean} liked
 * @prop {boolean} subscribed
 */

/**
 * @typedef {object} UpdatePost
 * @prop {string=} content
 * @prop {boolean=} nsfw
 * @prop {string=} spoilerOf
 */

/**
 * @typedef {object} UpdatedPost
 * @prop {string} content
 * @prop {boolean} nsfw
 * @prop {string} spoilerOf
 * @prop {string|Date} UpdatedAt
 */

/**
 * @typedef {object} UpdateComment
 * @prop {string=} content
 */

/**
 * @typedef {object} UpdatedComment
 * @prop {string} content
 */

/**
 * @typedef {object} ReactionCount
 * @prop {string} type
 * @prop {string} reaction
 * @prop {number} count
 */

/**
 * @typedef {object} TimelineItemExt
 * @prop {string} timelineItemID
 */

/**
 * @template T
 * @typedef Page
 * @prop {T[]} items
 * @prop {string|null} startCursor
 * @prop {string|null} endCursor
 */

/**
 * @typedef Comment
 * @prop {string} id
 * @prop {string} content
 * @prop {ReactionCount[]} reactions
 * @prop {string|Date} createdAt
 * @prop {User=} user
 * @prop {boolean} mine
 * @prop {boolean} liked
 */

/**
 * @typedef CreatePostInput
 * @prop {string} content
 * @prop {boolean=} NSFW
 * @prop {string=} spoilerOf
 */

/**
 * @typedef ToggleFollowOutput
 * @prop {number} followersCount
 * @prop {boolean} following
 */

/**
 * @typedef ToggleSubscriptionOutput
 * @prop {boolean} subscribed
 */

/**
 * @typedef Notification
 * @prop {string} id
 * @prop {string[]} actors
 * @prop {"follow"|"comment"|"post_mention"|"comment_mention"} type
 * @prop {string=} postID
 * @prop {boolean} read
 * @prop {string|Date} issuedAt
 */

export default undefined
