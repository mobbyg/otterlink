# Qt Chat service

The Qt client presents the shared Otter Link community chat as the first fully developed service view beyond the initial dashboard shell.

The current service is intentionally backed by the existing `/api/chat` endpoints. The client provides:

- a dedicated Lounge presentation inside the service shell
- Enter-to-send and an explicit Send action
- a 500-character client-side message limit
- a scrollable, alternating-row conversation view
- automatic following of new messages when already at the bottom
- preservation of the user's reading position when refreshing every five seconds

This is a foundation for future chat rooms, participant information, private messaging, and richer message presentation. Those features should be added as service capabilities rather than tying the Qt client directly to storage or database details.
