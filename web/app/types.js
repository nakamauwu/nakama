/**
 * @typedef {object} User
 * @prop {string} id
 * @prop {string} username
 * @prop {string|null} avatarURL
 */

/**
 * @typedef {object} ReactionCount
 * @prop {string} type
 * @prop {string} reaction
 * @prop {number} count
 */

/**
 * @typedef {object} Post
 * @prop {string} id
 * @prop {string} content
 * @prop {ReactionCount[]} reactionCounts
 * @prop {number} repliesCount
 * @prop {Date} createdAt
 * @prop {User|null} user
 */

export default undefined
