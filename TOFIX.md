Fix:

etails:
[
  {
    "@type": "type.googleapis.com/google.rpc.ErrorInfo",
    "domain": "googleapis.com",
    "metadata": {
      "consumer": "projects/215879942562",
      "quota_limit": "defaultPerDayPerProject",
      "quota_limit_value": "50000",
      "quota_location": "global",
      "quota_metric": "tasks.googleapis.com/default",
      "quota_unit": "1/d/{project}",
      "service": "tasks.googleapis.com"
    },
    "reason": "RATE_LIMIT_EXCEEDED"
  },
  {
    "@type": "type.googleapis.com/google.rpc.Help",
    "links": [
      {
        "description": "Request a higher quota limit.",
        "url": "https://cloud.google.com/docs/quotas/help/request_increase"
      }
    ]
  }
]
, rateLimitExceeded


---

Add a cleaner method to remove task in completed with more than a year old

