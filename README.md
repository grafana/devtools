# goals with this repo?

Download events from github api about one repo.

- Find new contributors.
  - follow up flagged contributors
  - see if flagged contributors keep providing.
  - 
  
- define SLA / SLI / SLO
  - What do we need for that? 
  
- find issues/PR's without a response.
- open, closed issues
- open, closed PR's
- includuing/excluding core team. 


# TODO
- get all events
  - get data from https://developer.github.com/v3/activity/events/#list-issue-events-for-a-repository
  - store each event in postgre
  - create transformation that can be rerunned.
