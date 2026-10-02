const receiverURL = "http://127.0.0.1:8765/search";
const googleDomains = ["google.com", "google.co.uk", "google.ca", "google.com.au", "google.co.in"];

function matchesDomain(host, domain) {
  return host === domain || host.endsWith(`.${domain}`);
}

function readSearch(urlString) {
  const url = new URL(urlString);
  const host = url.hostname.toLowerCase();
  const path = url.pathname.toLowerCase();
  let engine;
  let queryParameter;

  if (host === "search.yahoo.com" && path.startsWith("/search")) {
    engine = "Yahoo";
    queryParameter = "p";
  } else if ((host === "duckduckgo.com" || host.endsWith(".duckduckgo.com")) && path === "/") {
    engine = "DuckDuckGo";
    queryParameter = "q";
  } else if ((host === "bing.com" || host.endsWith(".bing.com")) && path === "/search") {
    engine = "Bing";
    queryParameter = "q";
  } else if ((host === "youtube.com" || host.endsWith(".youtube.com")) && path === "/results") {
    engine = "YouTube";
    queryParameter = "search_query";
  } else if (googleDomains.some((domain) => matchesDomain(host, domain)) && path === "/search") {
    engine = "Google";
    queryParameter = "q";
  } else {
    return null;
  }

  const query = url.searchParams.get(queryParameter)?.trim();
  return query ? { query, engine } : null;
}

chrome.webNavigation.onCommitted.addListener(async (details) => {
  if (details.frameId !== 0) return;

  let search;
  let tab;
  try {
    search = readSearch(details.url);
    if (!search) return;
    tab = await chrome.tabs.get(details.tabId);
  } catch (error) {
    console.warn("Could not inspect committed search navigation", error);
    return;
  }

  try {
    const response = await fetch(receiverURL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        ...search,
        incognito: Boolean(tab.incognito),
        ...(tab.title ? { window_title: tab.title } : {})
      })
    });
    if (!response.ok) throw new Error(`receiver returned HTTP ${response.status}`);
    await chrome.action.setBadgeText({ text: "OK" });
    await chrome.action.setBadgeBackgroundColor({ color: "#16794b" });
  } catch (error) {
    console.error("Could not report submitted search to local monitor", error);
    await chrome.action.setBadgeText({ text: "!" });
    await chrome.action.setBadgeBackgroundColor({ color: "#b42318" });
  }
});
