// Adapted from the OKF bundle viewer in GoogleCloudPlatform/knowledge-catalog (Apache-2.0). See NOTICE.
// Changes: staleness is computed when the page is viewed, so a committed page stays the same across
// days; relative and bundle-relative links both open their concept; a source that names a concept
// draws a dashed edge and opens that concept; nodes take their layer's color, with a legend;
// summary and attribution show when present.
(function () {
  const bundle = window.BUNDLE;
  const now = Date.now();
  for (const n of bundle.nodes) {
    const sa = n.data.stale_after ? Date.parse(n.data.stale_after) : NaN;
    n.data.stale = !Number.isNaN(sa) && /T/.test(n.data.stale_after) && now >= sa;
  }
  const bundleName = window.BUNDLE_NAME;
  document.title = `${bundleName} — OKF Viewer`;
  document.getElementById("bundle-name").textContent = bundleName;

  // Populate type filter
  const typeSelect = document.getElementById("filter-type");
  for (const t of bundle.types) {
    const opt = document.createElement("option");
    opt.value = t;
    opt.textContent = t;
    typeSelect.appendChild(opt);
  }

  // Layer legend: a click dims every other layer, and a second click clears it.
  const legend = document.getElementById("legend");
  let activeLayer = "";
  for (const layer of bundle.layers || []) {
    const item = document.createElement("span");
    item.className = "legend-item";
    const swatch = document.createElement("span");
    swatch.className = "legend-swatch";
    swatch.style.background = layer.color;
    item.appendChild(swatch);
    item.appendChild(document.createTextNode(layer.name));
    item.addEventListener("click", () => {
      activeLayer = activeLayer === layer.name ? "" : layer.name;
      dimBy((n) => activeLayer && n.data("layer") !== activeLayer);
    });
    legend.appendChild(item);
  }

  // Build reverse-link index for backlinks
  const backlinks = {};
  for (const edge of bundle.edges) {
    const { source, target } = edge.data;
    if (!(backlinks[target] ||= []).includes(source)) backlinks[target].push(source);
  }

  // Look up node label/type by id
  const nodeIndex = {};
  for (const n of bundle.nodes) nodeIndex[n.data.id] = n.data;

  const cy = cytoscape({
    container: document.getElementById("graph"),
    elements: [...bundle.nodes, ...bundle.edges],
    style: [
      {
        selector: "node",
        style: {
          "background-color": "data(color)",
          "label": "data(label)",
          "color": "#0f172a",
          "font-size": 11,
          "text-valign": "bottom",
          "text-margin-y": 4,
          "text-wrap": "wrap",
          "text-max-width": 120,
          "width": "data(size)",
          "height": "data(size)",
          "border-width": 1,
          "border-color": "#0f172a",
        },
      },
      {
        selector: "node[?stale]",
        style: {
          "border-width": 2,
          "border-color": "#b91c1c",
          "border-style": "dashed",
        },
      },
      {
        selector: 'node[status = "deprecated"]',
        style: {
          "opacity": 0.55,
        },
      },
      {
        selector: "node:selected",
        style: {
          "border-width": 3,
          "border-color": "#f59e0b",
        },
      },
      {
        selector: "edge",
        style: {
          "width": 1.5,
          "line-color": "#cbd5e1",
          "target-arrow-color": "#cbd5e1",
          "target-arrow-shape": "triangle",
          "curve-style": "bezier",
          "arrow-scale": 0.9,
        },
      },
      {
        selector: 'edge[kind = "source"]',
        style: {
          "line-style": "dashed",
          "line-color": "#94a3b8",
          "target-arrow-color": "#94a3b8",
        },
      },
      {
        selector: "edge:selected",
        style: {
          "line-color": "#f59e0b",
          "target-arrow-color": "#f59e0b",
          "width": 2.5,
        },
      },
      {
        selector: ".dim",
        style: { "opacity": 0.15 },
      },
    ],
    layout: { name: "cose", animate: false, padding: 30 },
    wheelSensitivity: 0.2,
  });

  cy.on("tap", "node", (evt) => showDetail(evt.target.id()));
  cy.on("tap", (evt) => {
    if (evt.target === cy) clearSelection();
  });

  document.getElementById("layout").addEventListener("change", (e) => {
    cy.layout({ name: e.target.value, animate: false, padding: 30 }).run();
  });

  document.getElementById("reset").addEventListener("click", () => {
    cy.fit(null, 30);
    clearSelection();
  });

  document.getElementById("search").addEventListener("input", (e) => {
    const q = e.target.value.trim().toLowerCase();
    if (!q) {
      cy.elements().removeClass("dim");
      return;
    }
    cy.nodes().forEach((n) => {
      const d = n.data();
      const hay =
        (d.label || "").toLowerCase() + " " +
        d.id.toLowerCase() + " " +
        (d.tags || []).join(" ").toLowerCase();
      n.toggleClass("dim", !hay.includes(q));
    });
    cy.edges().forEach((edge) => {
      const src = edge.source();
      const tgt = edge.target();
      edge.toggleClass("dim", src.hasClass("dim") || tgt.hasClass("dim"));
    });
  });

  document.getElementById("filter-type").addEventListener("change", (e) => {
    const t = e.target.value;
    dimBy((n) => t && n.data("type") !== t);
  });

  function dimBy(hide) {
    cy.nodes().forEach((n) => n.toggleClass("dim", Boolean(hide(n))));
    cy.edges().forEach((edge) => {
      edge.toggleClass("dim", edge.source().hasClass("dim") || edge.target().hasClass("dim"));
    });
  }

  function clearSelection() {
    cy.elements().unselect();
    document.getElementById("detail-empty").hidden = false;
    document.getElementById("detail-content").hidden = true;
  }

  // showDetail fills the detail pane. focus centers the graph on the concept; the first view leaves
  // it unset, so the whole graph stays in frame.
  function showDetail(conceptId, focus = true) {
    const data = nodeIndex[conceptId];
    if (!data) return;
    cy.elements().unselect();
    const node = cy.getElementById(conceptId);
    if (node) node.select();

    document.getElementById("detail-empty").hidden = true;
    const content = document.getElementById("detail-content");
    content.hidden = false;

    const chip = document.getElementById("detail-type");
    chip.textContent = data.type;
    chip.style.background = data.color;

    document.getElementById("detail-title").textContent = data.label;
    document.getElementById("detail-id").textContent = conceptId;
    document.getElementById("detail-description").textContent = data.description || "—";
    const summaryEl = document.getElementById("detail-summary");
    summaryEl.textContent = data.summary || "";
    summaryEl.hidden = document.getElementById("detail-summary-label").hidden = !data.summary;

    const resourceEl = document.getElementById("detail-resource");
    resourceEl.innerHTML = "";
    if (data.resource) {
      const a = document.createElement("a");
      a.href = data.resource;
      a.textContent = data.resource;
      a.target = "_blank";
      a.rel = "noopener";
      a.className = "external";
      resourceEl.appendChild(a);
    } else {
      resourceEl.textContent = "—";
    }

    const tagsEl = document.getElementById("detail-tags");
    tagsEl.innerHTML = "";
    if (data.tags && data.tags.length) {
      for (const t of data.tags) {
        const span = document.createElement("span");
        span.className = "tag";
        span.textContent = t;
        tagsEl.appendChild(span);
      }
    } else {
      tagsEl.textContent = "—";
    }

    // v0.2 signal badges: status, trust tier, staleness.
    const badgesEl = document.getElementById("detail-badges");
    badgesEl.innerHTML = "";
    const status = data.status || "stable";
    badgesEl.appendChild(makeBadge(status, "status-" + status));
    const tier = data.trust_tier || "unverified";
    badgesEl.appendChild(makeBadge(tier.replace(/-/g, " "), "trust-" + tier));
    if (data.stale) {
      const label = data.stale_after ? `stale (since ${data.stale_after})` : "stale";
      badgesEl.appendChild(makeBadge(label, "stale"));
    } else if (data.stale_after) {
      badgesEl.appendChild(makeBadge(`stale after ${data.stale_after}`, "fresh"));
    }

    document.getElementById("detail-generated").textContent = formatActorEvent(data.generated);

    const verifiedEl = document.getElementById("detail-verified");
    const verified = data.verified || [];
    if (verified.length) {
      verifiedEl.textContent = verified.map(formatActorEvent).join("; ");
    } else {
      verifiedEl.textContent = "—";
    }

    const sourcesEl = document.getElementById("detail-sources");
    sourcesEl.innerHTML = "";
    const sources = data.sources || [];
    if (sources.length) {
      const ul = document.createElement("ul");
      ul.className = "sources-list";
      for (const s of sources) {
        const li = document.createElement("li");
        const label = s.title || s.resource || s.id || "source";
        const target = s.resource && !/^[a-z][a-z0-9+.-]*:/i.test(s.resource) ? resolve(conceptId, s.resource) : null;
        if (target && nodeIndex[target]) {
          const a = document.createElement("a");
          a.className = "internal";
          a.href = "javascript:void(0)";
          a.textContent = s.title ? `${s.title} (${s.resource})` : s.resource;
          a.addEventListener("click", () => showDetail(target));
          li.appendChild(a);
        } else if (s.resource && /^https?:\/\//.test(s.resource)) {
          const a = document.createElement("a");
          a.href = s.resource;
          a.textContent = label;
          a.target = "_blank";
          a.rel = "noopener";
          a.className = "external";
          li.appendChild(a);
        } else {
          li.textContent = s.resource && s.resource !== label ? `${label} (${s.resource})` : label;
        }
        ul.appendChild(li);
      }
      sourcesEl.appendChild(ul);
    } else {
      sourcesEl.textContent = "—";
    }

    const attrLabel = document.getElementById("detail-attribution-label");
    const attrEl = document.getElementById("detail-attribution");
    attrEl.innerHTML = "";
    const attr = data.attribution;
    attrLabel.hidden = attrEl.hidden = !attr;
    if (attr) {
      const ul = document.createElement("ul");
      ul.className = "attribution-list";
      for (const key of ["derivation", "model", "authors", "contributors"]) {
        if (attr[key] === undefined) continue;
        const li = document.createElement("li");
        const value = Array.isArray(attr[key]) ? attr[key].join(", ") : String(attr[key]);
        li.textContent = `${key}: ${value}`;
        ul.appendChild(li);
      }
      attrEl.appendChild(ul);
    }

    const body = bundle.bodies[conceptId] || "";
    const html = marked.parse(body, { breaks: false, gfm: true });
    const bodyEl = document.getElementById("detail-body");
    bodyEl.innerHTML = html;
    rewriteInternalLinks(bodyEl, conceptId);

    const bl = backlinks[conceptId] || [];
    const blSection = document.getElementById("detail-backlinks");
    const blList = document.getElementById("backlinks-list");
    blList.innerHTML = "";
    if (bl.length) {
      blSection.hidden = false;
      for (const src of bl) {
        const li = document.createElement("li");
        const a = document.createElement("a");
        a.textContent = nodeIndex[src]?.label || src;
        a.dataset.target = src;
        a.addEventListener("click", () => showDetail(src));
        li.appendChild(a);
        const muted = document.createElement("span");
        muted.className = "muted";
        muted.textContent = ` (${src})`;
        li.appendChild(muted);
        blList.appendChild(li);
      }
    } else {
      blSection.hidden = true;
    }

    if (focus) cy.animate({ center: { eles: node }, zoom: Math.max(cy.zoom(), 1.0) }, { duration: 200 });
  }

  function makeBadge(text, cls) {
    const span = document.createElement("span");
    span.className = "badge " + cls;
    span.textContent = text;
    return span;
  }

  function formatActorEvent(event) {
    if (!event || !event.by) return "—";
    return event.at ? `${event.by} · ${event.at}` : String(event.by);
  }

  // resolve turns a link in the concept fromId into a concept id, for bundle-relative and relative
  // links alike.
  function resolve(fromId, href) {
    const path = href.split("#")[0];
    if (!path.endsWith(".md")) return null;
    const parts = path.startsWith("/") ? [] : fromId.split("/").slice(0, -1);
    for (const seg of path.replace(/^\//, "").split("/")) {
      if (seg === "" || seg === ".") continue;
      if (seg === "..") parts.pop();
      else parts.push(seg);
    }
    return parts.join("/").slice(0, -3);
  }

  function rewriteInternalLinks(root, fromId) {
    root.querySelectorAll("a[href]").forEach((a) => {
      const href = a.getAttribute("href");
      if (!href) return;
      if (!/^[a-z][a-z0-9+.-]*:/i.test(href) && !href.startsWith("#")) {
        const target = resolve(fromId, href);
        if (target && nodeIndex[target]) {
          a.className = "internal";
          a.setAttribute("href", "javascript:void(0)");
          a.addEventListener("click", (e) => {
            e.preventDefault();
            showDetail(target);
          });
          return;
        }
      }
      a.className = "external";
      a.setAttribute("target", "_blank");
      a.setAttribute("rel", "noopener");
    });
  }

  // Auto-show the first concept.
  const initial = bundle.nodes[0];
  if (initial) showDetail(initial.data.id, false);
})();
