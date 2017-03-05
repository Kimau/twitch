package twitch

// Various prefixes extracted from RFC1459.
const (
	IrcPrefixChannel     = '#' // Normal channel
	IrcPrefixDistributed = '&' // Distributed channel

	IrcPrefixOwner        = '~' // Channel owner +q (non-standard)
	IrcPrefixAdmin        = '&' // Channel admin +a (non-standard)
	IrcPrefixOperator     = '@' // Channel operator +o
	IrcPrefixHalfOperator = '%' // Channel half operator +h (non-standard)
	IrcPrefixVoice        = '+' // User has voice +v
)

// User modes as defined by RFC1459 section 4.2.3.2.
const (
	IrcUserModeInvisible     = 'i' // User is invisible
	IrcUserModeServerNotices = 's' // User wants to receive server notices
	IrcUserModeWallops       = 'w' // User wants to receive Wallops
	IrcUserModeOperator      = 'o' // Server operator
)

// Channel modes as defined by RFC1459 section 4.2.3.1
const (
	IrcModeOperator   = 'o' // Operator privileges
	IrcModeVoice      = 'v' // Ability to speak on a moderated channel
	IrcModePrivate    = 'p' // Private channel
	IrcModeSecret     = 's' // Secret channel
	IrcModeInviteOnly = 'i' // Users can't join without invite
	IrcModeTopic      = 't' // Topic can only be set by an operator
	IrcModeModerated  = 'm' // Only voiced users and operators can talk
	IrcModeLimit      = 'l' // User limit
	IrcModeKey        = 'k' // Channel password

	IrcModeOwner        = 'q' // Owner privileges (non-standard)
	IrcModeAdmin        = 'a' // Admin privileges (non-standard)
	IrcModeHalfOperator = 'h' // Half-operator privileges (non-standard)
)

// IRC commands extracted from RFC2812 section 3 and RFC2813 section 4.
const (
	IrcCmdPass     = "PASS"
	IrcCmdNick     = "NICK"
	IrcCmdUser     = "USER"
	IrcCmdOper     = "OPER"
	IrcCmdMode     = "MODE"
	IrcCmdService  = "SERVICE"
	IrcCmdQuit     = "QUIT"
	IrcCmdSquit    = "SQUIT"
	IrcCmdJoin     = "JOIN"
	IrcCmdPart     = "PART"
	IrcCmdTopic    = "TOPIC"
	IrcCmdNames    = "NAMES"
	IrcCmdList     = "LIST"
	IrcCmdInvite   = "INVITE"
	IrcCmdKick     = "KICK"
	IrcCmdPrivmsg  = "PRIVMSG"
	IrcCmdNotice   = "NOTICE"
	IrcCmdMotd     = "MOTD"
	IrcCmdLusers   = "LUSERS"
	IrcCmdVersion  = "VERSION"
	IrcCmdStats    = "STATS"
	IrcCmdLinks    = "LINKS"
	IrcCmdTime     = "TIME"
	IrcCmdConnect  = "CONNECT"
	IrcCmdTrace    = "TRACE"
	IrcCmdAdmin    = "ADMIN"
	IrcCmdInfo     = "INFO"
	IrcCmdServlist = "SERVLIST"
	IrcCmdSquery   = "SQUERY"
	IrcCmdWho      = "WHO"
	IrcCmdWhois    = "WHOIS"
	IrcCmdWhowas   = "WHOWAS"
	IrcCmdKill     = "KILL"
	IrcCmdPing     = "PING"
	IrcCmdPong     = "PONG"
	IrcCmdError    = "ERROR"
	IrcCmdAway     = "AWAY"
	IrcCmdRehash   = "REHASH"
	IrcCmdDie      = "DIE"
	IrcCmdRestart  = "RESTART"
	IrcCmdSummon   = "SUMMON"
	IrcCmdUsers    = "USERS"
	IrcCmdWallops  = "WALLOPS"
	IrcCmdUserhost = "USERHOST"
	IrcCmdIson     = "ISON"
	IrcCmdServer   = "SERVER"
	IrcCmdNjoin    = "NJOIN"
)

// Numeric IRC replies extracted from RFC2812 section 5.
const (
	IrcReplyWelcome           = "001"
	IrcReplyYourhost          = "002"
	IrcReplyCreated           = "003"
	IrcReplyMyinfo            = "004"
	IrcReplyBounce            = "005"
	IrcReplyIsupport          = "005"
	IrcReplyUserhost          = "302"
	IrcReplyIson              = "303"
	IrcReplyAway              = "301"
	IrcReplyUnaway            = "305"
	IrcReplyNowaway           = "306"
	IrcReplyWhoisuser         = "311"
	IrcReplyWhoisserver       = "312"
	IrcReplyWhoisoperator     = "313"
	IrcReplyWhoisidle         = "317"
	IrcReplyEndofwhois        = "318"
	IrcReplyWhoischannels     = "319"
	IrcReplyWhowasuser        = "314"
	IrcReplyEndofwhowas       = "369"
	IrcReplyListstart         = "321"
	IrcReplyList              = "322"
	IrcReplyListend           = "323"
	IrcReplyUniqopis          = "325"
	IrcReplyChannelmodeis     = "324"
	IrcReplyNotopic           = "331"
	IrcReplyTopic             = "332"
	IrcReplyInviting          = "341"
	IrcReplySummoning         = "342"
	IrcReplyInvitelist        = "346"
	IrcReplyEndofinvitelist   = "347"
	IrcReplyExceptlist        = "348"
	IrcReplyEndofexceptlist   = "349"
	IrcReplyVersion           = "351"
	IrcReplyWhoreply          = "352"
	IrcReplyEndofwho          = "315"
	IrcReplyNamreply          = "353"
	IrcReplyEndofnames        = "366"
	IrcReplyLinks             = "364"
	IrcReplyEndoflinks        = "365"
	IrcReplyBanlist           = "367"
	IrcReplyEndofbanlist      = "368"
	IrcReplyInfo              = "371"
	IrcReplyEndofinfo         = "374"
	IrcReplyMotdstart         = "375"
	IrcReplyMotd              = "372"
	IrcReplyEndofmotd         = "376"
	IrcReplyYoureoper         = "381"
	IrcReplyRehashing         = "382"
	IrcReplyYoureservice      = "383"
	IrcReplyTime              = "391"
	IrcReplyUsersstart        = "392"
	IrcReplyUsers             = "393"
	IrcReplyEndofusers        = "394"
	IrcReplyNousers           = "395"
	IrcReplyTracelink         = "200"
	IrcReplyTraceconnecting   = "201"
	IrcReplyTracehandshake    = "202"
	IrcReplyTraceunknown      = "203"
	IrcReplyTraceoperator     = "204"
	IrcReplyTraceuser         = "205"
	IrcReplyTraceserver       = "206"
	IrcReplyTraceservice      = "207"
	IrcReplyTracenewtype      = "208"
	IrcReplyTraceclass        = "209"
	IrcReplyTracereconnect    = "210"
	IrcReplyTracelog          = "261"
	IrcReplyTraceend          = "262"
	IrcReplyStatslinkinfo     = "211"
	IrcReplyStatscommands     = "212"
	IrcReplyEndofstats        = "219"
	IrcReplyStatsuptime       = "242"
	IrcReplyStatsoline        = "243"
	IrcReplyUmodeis           = "221"
	IrcReplyServlist          = "234"
	IrcReplyServlistend       = "235"
	IrcReplyLuserclient       = "251"
	IrcReplyLuserop           = "252"
	IrcReplyLuserunknown      = "253"
	IrcReplyLuserchannels     = "254"
	IrcReplyLuserme           = "255"
	IrcReplyAdminme           = "256"
	IrcReplyAdminloc1         = "257"
	IrcReplyAdminloc2         = "258"
	IrcReplyAdminemail        = "259"
	IrcReplyTryagain          = "263"
	IrcErrorNosuchnick        = "401"
	IrcErrorNosuchserver      = "402"
	IrcErrorNosuchchannel     = "403"
	IrcErrorCannotsendtochan  = "404"
	IrcErrorToomanychannels   = "405"
	IrcErrorWasnosuchnick     = "406"
	IrcErrorToomanytargets    = "407"
	IrcErrorNosuchservice     = "408"
	IrcErrorNoorigin          = "409"
	IrcErrorNorecipient       = "411"
	IrcErrorNotexttosend      = "412"
	IrcErrorNotoplevel        = "413"
	IrcErrorWildtoplevel      = "414"
	IrcErrorBadmask           = "415"
	IrcErrorUnknowncommand    = "421"
	IrcErrorNomotd            = "422"
	IrcErrorNoadmininfo       = "423"
	IrcErrorFileerror         = "424"
	IrcErrorNonicknamegiven   = "431"
	IrcErrorErroneusnickname  = "432"
	IrcErrorNicknameinuse     = "433"
	IrcErrorNickcollision     = "436"
	IrcErrorUnavailresource   = "437"
	IrcErrorUsernotinchannel  = "441"
	IrcErrorNotonchannel      = "442"
	IrcErrorUseronchannel     = "443"
	IrcErrorNologin           = "444"
	IrcErrorSummondisabled    = "445"
	IrcErrorUsersdisabled     = "446"
	IrcErrorNotregistered     = "451"
	IrcErrorNeedmoreparams    = "461"
	IrcErrorAlreadyregistred  = "462"
	IrcErrorNopermforhost     = "463"
	IrcErrorPasswdmismatch    = "464"
	IrcErrorYourebannedcreep  = "465"
	IrcErrorYouwillbebanned   = "466"
	IrcErrorKeyset            = "467"
	IrcErrorChannelisfull     = "471"
	IrcErrorUnknownmode       = "472"
	IrcErrorInviteonlychan    = "473"
	IrcErrorBannedfromchan    = "474"
	IrcErrorBadchannelkey     = "475"
	IrcErrorBadchanmask       = "476"
	IrcErrorNochanmodes       = "477"
	IrcErrorBanlistfull       = "478"
	IrcErrorNoprivileges      = "481"
	IrcErrorChanoprivsneeded  = "482"
	IrcErrorCantkillserver    = "483"
	IrcErrorRestricted        = "484"
	IrcErrorUniqopprivsneeded = "485"
	IrcErrorNooperhost        = "491"
	IrcErrorUmodeunknownflag  = "501"
	IrcErrorUsersdontmatch    = "502"
)

// IRC commands extracted from the IRCv3 spec at http://www.ircv3.org/.
const (
	IrcCap      = "CAP"
	IrcCapLs    = "LS"    // Subcommand (param)
	IrcCapList  = "LIST"  // Subcommand (param)
	IrcCapReq   = "REQ"   // Subcommand (param)
	IrcCapAck   = "ACK"   // Subcommand (param)
	IrcCapNak   = "NAK"   // Subcommand (param)
	IrcCapClear = "CLEAR" // Subcommand (param)
	IrcCapEnd   = "END"   // Subcommand (param)

	AUTHENTICATE = "AUTHENTICATE"
)

// Numeric IRC replies extracted from the IRCv3 spec.
const (
	IrcReplyLoggedin    = "900"
	IrcReplyLoggedout   = "901"
	IrcReplyNicklocked  = "902"
	IrcReplySaslsuccess = "903"
	IrcErrorSaslfail    = "904"
	IrcErrorSasltoolong = "905"
	IrcErrorSaslaborted = "906"
	IrcErrorSaslalready = "907"
	IrcErrorSaslmechs   = "908"
)

// RFC2812, section 5.3
const (
	IrcReplyStatscline    = "213"
	IrcReplyStatsnline    = "214"
	IrcReplyStatsiline    = "215"
	IrcReplyStatskline    = "216"
	IrcReplyStatsqline    = "217"
	IrcReplyStatsyline    = "218"
	IrcReplyServiceinfo   = "231"
	IrcReplyEndofservices = "232"
	IrcReplyService       = "233"
	IrcReplyStatsvline    = "240"
	IrcReplyStatslline    = "241"
	IrcReplyStatshline    = "244"
	IrcReplyStatssline    = "245"
	IrcReplyStatsping     = "246"
	IrcReplyStatsbline    = "247"
	IrcReplyStatsdline    = "250"
	IrcReplyNone          = "300"
	IrcReplyWhoischanop   = "316"
	IrcReplyKilldone      = "361"
	IrcReplyClosing       = "362"
	IrcReplyCloseend      = "363"
	IrcReplyInfostart     = "373"
	IrcReplyMyportis      = "384"
	IrcErrorNoservicehost = "492"
)

// Other constants
const (
	IrcErrorToomanymatches = "416" // Used on IRCNet
	IrcReplyTopicwhotime   = "333" // From ircu, in use on Freenode
	IrcReplyLocalusers     = "265" // From aircd, Hybrid, Hybrid, Bahamut, in use on Freenode
	IrcReplyGlobalusers    = "266" // From aircd, Hybrid, Hybrid, Bahamut, in use on Freenode
)

// Twitch Tagged Commands
const (
	TwitchCmdClearChat       = "CLEARCHAT"       // Temporary or permanent ban on a channel.
	TwitchCmdGlobalUserState = "GLOBALUSERSTATE" // On successful login.
	TwitchCmdRoomState       = "ROOMSTATE"       // When a user joins a channel or a room setting is changed.
	TwitchCmdUserNotice      = "USERNOTICE"      // On re-subscription to a channel.
	TwitchCmdUserState       = "USERSTATE"       // When a user joins a channel or sends a PRIVMSG to a channel.
	TwitchCmdHostTarget      = "HOSTTARGET"      // Host starts or stops a message.
	TwitchCmdReconnect       = "RECONNECT"       // Rejoin channels after a restart.
)

// Twitch Msg ID Tages
const (
	TwitchMsgAlreadyBanned       = "already_banned"         // <user> is already banned in this room.
	TwitchMsgAlreadyEmoteOnlyOff = "already_emote_only_off" // This room is not in emote-only mode.
	TwitchMsgAlreadyEmoteOnlyOn  = "already_emote_only_on"  // This room is already in emote-only mode.
	TwitchMsgAlreadyR9kOff       = "already_r9k_off"        // This room is not in r9k mode.
	TwitchMsgAlreadyR9kOn        = "already_r9k_on"         // This room is already in r9k mode.
	TwitchMsgAlreadySubsOff      = "already_subs_off"       // This room is not in subscribers-only mode.
	TwitchMsgAlreadySubsOn       = "already_subs_on"        // This room is already in subscribers-only mode.
	TwitchMsgBadHostHosting      = "bad_host_hosting"       // This channel is hosting <channel>.
	TwitchMsgBadUnbanNoBan       = "bad_unban_no_ban"       // <user> is not banned from this room.
	TwitchMsgBanSuccess          = "ban_success"            // <user> is banned from this room.
	TwitchMsgEmoteOnlyOff        = "emote_only_off"         // This room is no longer in emote-only mode.
	TwitchMsgEmoteOnlyOn         = "emote_only_on"          // This room is now in emote-only mode.
	TwitchMsgHostOff             = "host_off"               // Exited host mode.
	TwitchMsgHostOn              = "host_on"                // Now hosting <channel>.
	TwitchMsgHostsRemaining      = "hosts_remaining"        // There are <number> host commands remaining this half hour.
	TwitchMsgMsgChannelSuspended = "msg_channel_suspended"  // This channel is suspended.
	TwitchMsgR9kOff              = "r9k_off"                // This room is no longer in r9k mode.
	TwitchMsgR9kOn               = "r9k_on"                 // This room is now in r9k mode.
	TwitchMsgSlowOff             = "slow_off"               // This room is no longer in slow mode.
	TwitchMsgSlowOn              = "slow_on"                // This room is now in slow mode. You may send messages every <slow seconds> seconds.
	TwitchMsgSubsOff             = "subs_off"               // This room is no longer in subscribers-only mode.
	TwitchMsgSubsOn              = "subs_on"                // This room is now in subscribers-only mode.
	TwitchMsgTimeoutSuccess      = "timeout_success"        // <user> has been timed out for <duration> seconds.
	TwitchMsgUnbanSuccess        = "unban_success"          // <user> is no longer banned from this chat room.
	TwitchMsgUnrecognizedCmd     = "unrecognized_cmd"       // Unrecognized command: <command>
)

// Twitch Tags
const (
	TwitchTagUserId            = "id"
	TwitchTagUserBadge         = "badges"
	TwitchTagUserColor         = "color"
	TwitchTagUserDisplayName   = "display-name"
	TwitchTagUserEmoteSet      = "emote-sets"
	TwitchTagUserMod           = "mod"
	TwitchTagUserSub           = "subscriber"
	TwitchTagUserType          = "user-type"
	TwitchTagUserTurbo         = "turbo"
	TwitchTagRoomId            = "room-id"
	TwitchTagRoomFollowersOnly = "followers-only"
	TwitchTagRoomR9K           = "r9k"
	TwitchTagRoomSlow          = "slow"
	TwitchTagRoomSubOnly       = "subs-only"
	TwitchTagRoomLang          = "broadcaster-lang"
	TwitchTagRoomEmote         = "emote-only"
	TwitchTagBits              = "bits"
)

const (
	TwitchBadgeStaff       = "staff"
	TwitchBadgeTurbo       = "turbo"
	TwitchBadgeSub         = "sub"
	TwitchBadgeMod         = "mod"
	TwitchBadgeBroadcaster = "broadcaster"
)
