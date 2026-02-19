    const CARD_WIDTH = 380;
    const SOURCE_PREVIEW_LIMIT = 5;
    const PERMISSION_PREVIEW_LIMIT = 4;
    const CARD_BASE_HEIGHT = 54;
    const CARD_TITLE_LINE_HEIGHT = 18;
    const CARD_ID_LINE_HEIGHT = 12;
    const CARD_MIN_HEIGHT = 122;

    const runBtn = document.getElementById('run');
    const statusEl = document.getElementById('status');
    const statsEl = document.getElementById('stats');
    const warningsEl = document.getElementById('warnings');
    const gapsEl = document.getElementById('gaps');
    const resourceMapBody = document.getElementById('resourceMapBody');
    const rawRequestEl = document.getElementById('rawRequest');
    const rawResponseEl = document.getElementById('rawResponse');
    const rawResponseMetaEl = document.getElementById('rawResponseMeta');
    const copyRequestBtn = document.getElementById('copyRequest');
    const copyResponseBtn = document.getElementById('copyResponse');
    const requestViewEl = document.getElementById('requestView');
    const responseViewEl = document.getElementById('responseView');
    const showAggregateEdgesEl = document.getElementById('showAggregateEdges');
    const onlyReachableEl = document.getElementById('onlyReachable');
    const showPermissionsEl = document.getElementById('showPermissions');
    const showRolePermissionsEl = document.getElementById('showRolePermissions');
    const spreadEdgesEl = document.getElementById('spreadEdges');
    const focusModeEl = document.getElementById('focusMode');
    const includePodsEl = document.getElementById('includePods');
    const includeWorkloadsEl = document.getElementById('includeWorkloads');
    const filterPhantomAPIsEl = document.getElementById('filterPhantomAPIs');
    const podPhaseModeEl = document.getElementById('podPhaseMode');
    const maxPodsPerSubjectEl = document.getElementById('maxPodsPerSubject');
    const maxWorkloadsPerPodEl = document.getElementById('maxWorkloadsPerPod');
    const namespaceScopeStrictEl = document.getElementById('namespaceScopeStrict');
    const impersonateUserEl = document.getElementById('impersonateUser');
    const impersonateGroupEl = document.getElementById('impersonateGroup');
    const runtimeViewEl = document.getElementById('runtimeView');
    const laneSpacingEl = document.getElementById('laneSpacing');
    const rowSpacingEl = document.getElementById('rowSpacing');
    const autoFitEl = document.getElementById('autoFit');
    const fitGraphBtn = document.getElementById('fitGraph');
    const legendRunsAsTextEl = document.getElementById('legendRunsAsText');
    const legendOwnedByTextEl = document.getElementById('legendOwnedByText');
    const canvasHeightEl = document.getElementById('canvasHeight');
    const flowRootEl = document.getElementById('flowRoot');
    const discoverOptionsBtn = document.getElementById('discoverOptions');
    const clearSelectorChecksBtn = document.getElementById('clearSelectorChecks');
    const selectorSummaryEl = document.getElementById('selectorSummary');
    const selectorDiscoverStatusEl = document.getElementById('selectorDiscoverStatus');

    const selectorListVerbsEl = document.getElementById('selectorListVerbs');
    const selectorListAPIGroupsEl = document.getElementById('selectorListAPIGroups');
    const selectorListResourcesEl = document.getElementById('selectorListResources');
    const selectorListNonResourceURLsEl = document.getElementById('selectorListNonResourceURLs');
    const selectorCountVerbsEl = document.getElementById('selectorCountVerbs');
    const selectorCountAPIGroupsEl = document.getElementById('selectorCountAPIGroups');
    const selectorCountResourcesEl = document.getElementById('selectorCountResources');
    const selectorCountNonResourceURLsEl = document.getElementById('selectorCountNonResourceURLs');
    const selectorFilterVerbsEl = document.getElementById('selectorFilterVerbs');
    const selectorFilterAPIGroupsEl = document.getElementById('selectorFilterAPIGroups');
    const selectorFilterResourcesEl = document.getElementById('selectorFilterResources');
    const selectorFilterNonResourceURLsEl = document.getElementById('selectorFilterNonResourceURLs');

    let lastGraph = { nodes: [], edges: [] };
    let lastBaseModel = { nodes: [], edges: [] };
    let lastRenderModel = { nodes: [], edges: [] };
    let lastFocusNodeID = '';
    let lastStatus = null;
    let setFlowModel = null;
    let fitFlowView = null;

    const rawStore = {
      request: '',
      response: ''
    };
    const EDGE_HANDLE_POSITIONS = [8, 14, 20, 26, 32, 38, 44, 50, 56, 62, 68, 74, 80, 86, 92];
    const EDGE_BANDS = {
      aggregate: [0, 1, 2],
      structural: [3, 4, 5],
      permission: [6, 7, 8],
      ownership: [9, 10, 11],
      runtime: [12, 13, 14]
    };
    const SELECTOR_KINDS = ['verbs', 'apiGroups', 'resources', 'nonResourceURLs'];
    const SELECTOR_INPUT_IDS = {
      verbs: 'verbs',
      apiGroups: 'apiGroups',
      resources: 'resources',
      nonResourceURLs: 'nonResourceURLs'
    };
    const SELECTOR_LIST_ELS = {
      verbs: selectorListVerbsEl,
      apiGroups: selectorListAPIGroupsEl,
      resources: selectorListResourcesEl,
      nonResourceURLs: selectorListNonResourceURLsEl
    };
    const SELECTOR_COUNT_ELS = {
      verbs: selectorCountVerbsEl,
      apiGroups: selectorCountAPIGroupsEl,
      resources: selectorCountResourcesEl,
      nonResourceURLs: selectorCountNonResourceURLsEl
    };
    const SELECTOR_FILTER_ELS = {
      verbs: selectorFilterVerbsEl,
      apiGroups: selectorFilterAPIGroupsEl,
      resources: selectorFilterResourcesEl,
      nonResourceURLs: selectorFilterNonResourceURLsEl
    };
    const selectorOptionCatalog = {
      verbs: [],
      apiGroups: [],
      resources: [],
      nonResourceURLs: []
    };
    const selectorOptionSelection = {
      verbs: new Set(),
      apiGroups: new Set(),
      resources: new Set(),
      nonResourceURLs: new Set()
    };
    const selectorOptionFilters = {
      verbs: '',
      apiGroups: '',
      resources: '',
      nonResourceURLs: ''
    };
    const resourceDisplayLabels = new Map();

    function applyCanvasHeight(value, persist) {
      const parsed = Number(value);
      const height = Number.isFinite(parsed) && parsed >= 520 ? Math.round(parsed) : 900;
      document.documentElement.style.setProperty('--canvas-height', `${height}px`);
      if (canvasHeightEl && String(canvasHeightEl.value) !== String(height)) {
        canvasHeightEl.value = String(height);
      }
      if (persist) {
        try {
          window.localStorage.setItem('rbacgraph.canvasHeight', String(height));
        } catch {
          // ignore localStorage failures
        }
      }
    }

    function escapeHTML(value) {
      return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
    }

    // -- Role Modal (interactive wildcard expansion) -------------------------
    const roleModalEl = document.getElementById('roleModal');
    const roleModalTitleEl = document.getElementById('roleModalTitle');
    const roleModalOriginalEl = document.getElementById('roleModalOriginal');
    const roleModalExpandedEl = document.getElementById('roleModalExpanded');

    /** Mirrors CSS custom properties from :root in styles.css. */
    const COLORS = {
      accent:          '#0f766e',
      edgeGrants:      '#0f766e',
      edgeBinds:       '#475569',
      edgePermissions: '#2563eb',
      edgeRunsAs:      '#0ea5a4',
      edgeOwnedBy:     '#334155',
      edgeAggregates:  '#c2410c',
      permBg:          '#eff6ff',
      aggBg:           '#fff7ed',
      cardSourceBg:    '#fff1f2',
      cardPodBg:       '#f0fdfa',
      surfaceMuted:    '#f8fafc',
      line:            '#d6d3d1',
    };

    const RULE_COLORS = ['#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899'];
    let modalWildcardRefs = [];
    let modalConcreteRefs = [];
    let modalSelectedIndices = new Set();

    function isWildcard(value) { return value === '*'; }

    function formatGroupLabel(g) {
      return (!g || g === '' || g === '<core>') ? 'core' : g;
    }

    // -- Wildcard classification & grouping ----------------------------------
    function classifyWildcardType(ref) {
      const dims = [];
      if (isWildcard(ref.apiGroup)) dims.push('group');
      if (isWildcard(ref.resource)) dims.push('resource');
      if (isWildcard(ref.verb)) dims.push('verb');
      return dims.join('+') || 'none';
    }

    function wildcardTypeBadgeText(type) {
      const labels = {
        'group': 'Expanded: apiGroup',
        'resource': 'Expanded: resource',
        'verb': 'Expanded: verb',
        'group+resource': 'Expanded: apiGroup + resource',
        'group+verb': 'Expanded: apiGroup + verb',
        'resource+verb': 'Expanded: resource + verb',
        'group+resource+verb': 'Expanded: apiGroup + resource + verb',
      };
      return labels[type] || 'Expanded';
    }

    function fmtResource(r) {
      return (r.resource || '*') + (r.subresource ? '/' + r.subresource : '');
    }

    function groupExpandedRefs(refs, wildcardType) {
      if (!refs || refs.length === 0) return { type: 'empty' };
      switch (wildcardType) {
        case 'group': {
          const byGroup = new Map();
          for (const r of refs) {
            const g = formatGroupLabel(r.apiGroup);
            if (!byGroup.has(g)) byGroup.set(g, []);
            byGroup.get(g).push(r);
          }
          return { type: 'single', key: 'apiGroup', groups: byGroup };
        }
        case 'resource': {
          const byRes = new Map();
          for (const r of refs) {
            const res = fmtResource(r);
            if (!byRes.has(res)) byRes.set(res, []);
            byRes.get(res).push(r);
          }
          return { type: 'single', key: 'resource', groups: byRes };
        }
        case 'verb': {
          const verbs = [...new Set(refs.map(r => r.verb))].sort();
          return { type: 'flat', key: 'verb', items: verbs };
        }
        case 'group+resource': {
          const tree = new Map();
          for (const r of refs) {
            const g = formatGroupLabel(r.apiGroup);
            if (!tree.has(g)) tree.set(g, new Map());
            const res = fmtResource(r);
            const resMap = tree.get(g);
            if (!resMap.has(res)) resMap.set(res, []);
            resMap.get(res).push(r);
          }
          return { type: 'two-level', keys: ['apiGroup', 'resource'], tree };
        }
        case 'group+verb': {
          const tree = new Map();
          for (const r of refs) {
            const g = formatGroupLabel(r.apiGroup);
            if (!tree.has(g)) tree.set(g, new Set());
            tree.get(g).add(r.verb);
          }
          return { type: 'two-level-flat', keys: ['apiGroup', 'verb'], tree };
        }
        case 'resource+verb': {
          const tree = new Map();
          for (const r of refs) {
            const res = fmtResource(r);
            if (!tree.has(res)) tree.set(res, new Set());
            tree.get(res).add(r.verb);
          }
          return { type: 'two-level-flat', keys: ['resource', 'verb'], tree };
        }
        case 'group+resource+verb': {
          const tree = new Map();
          for (const r of refs) {
            const g = formatGroupLabel(r.apiGroup);
            if (!tree.has(g)) tree.set(g, new Map());
            const resMap = tree.get(g);
            const res = fmtResource(r);
            if (!resMap.has(res)) resMap.set(res, new Set());
            resMap.get(res).add(r.verb);
          }
          return { type: 'three-level', tree };
        }
        default:
          return { type: 'empty' };
      }
    }

    // -- Render grouped expansion tree ---------------------------------------
    function renderGroupedExpansion(grouped) {
      let html = '';
      switch (grouped.type) {
        case 'single':
          for (const [name, items] of grouped.groups) {
            html += `<div class="rf-expansion-tree-group">`;
            html += `<div class="rf-expansion-tree-group-header" aria-expanded="true" onclick="toggleTreeGroup(this)">`;
            html += `<span class="toggle-arrow">\u25BC</span>`;
            html += `<span>${escapeHTML(name)}</span>`;
            html += `<span class="group-count">(${items.length})</span></div>`;
            html += `<div class="rf-expansion-tree-items"><div class="rf-modal-expanded-pills">`;
            for (const ref of items) {
              const verb = escapeHTML(ref.verb || '*');
              const resource = escapeHTML(fmtResource(ref));
              html += `<span class="rf-modal-expanded-pill"><span class="pill-verb">${verb}</span> <span class="pill-resource">${resource}</span></span>`;
            }
            html += `</div></div></div>`;
          }
          break;

        case 'flat':
          html += `<div class="rf-modal-expanded-pills">`;
          for (const item of grouped.items) {
            html += `<span class="rf-modal-expanded-pill"><span class="pill-verb">${escapeHTML(item)}</span></span>`;
          }
          html += `</div>`;
          break;

        case 'two-level':
          for (const [l1, l2Map] of grouped.tree) {
            const cnt = [...l2Map.values()].reduce((s, a) => s + a.length, 0);
            html += `<div class="rf-expansion-tree-group">`;
            html += `<div class="rf-expansion-tree-group-header" aria-expanded="true" onclick="toggleTreeGroup(this)">`;
            html += `<span class="toggle-arrow">\u25BC</span>`;
            html += `<span>${escapeHTML(l1)}</span>`;
            html += `<span class="group-count">(${cnt})</span></div>`;
            html += `<div class="rf-expansion-tree-items">`;
            for (const [l2, items] of l2Map) {
              html += `<div style="margin-bottom:4px;">`;
              html += `<span style="font-size:0.72rem;font-weight:600;color:#64748b;">${escapeHTML(l2)}</span> `;
              html += `<span class="rf-modal-expanded-pills" style="display:inline-flex;">`;
              for (const ref of items) {
                html += `<span class="rf-modal-expanded-pill"><span class="pill-verb">${escapeHTML(ref.verb)}</span></span>`;
              }
              html += `</span></div>`;
            }
            html += `</div></div>`;
          }
          break;

        case 'two-level-flat':
          for (const [l1, l2Set] of grouped.tree) {
            const items = [...l2Set].sort();
            html += `<div class="rf-expansion-tree-group">`;
            html += `<div class="rf-expansion-tree-group-header" aria-expanded="true" onclick="toggleTreeGroup(this)">`;
            html += `<span class="toggle-arrow">\u25BC</span>`;
            html += `<span>${escapeHTML(l1)}</span>`;
            html += `<span class="group-count">(${items.length})</span></div>`;
            html += `<div class="rf-expansion-tree-items"><div class="rf-modal-expanded-pills">`;
            for (const item of items) {
              html += `<span class="rf-modal-expanded-pill"><span class="pill-verb">${escapeHTML(item)}</span></span>`;
            }
            html += `</div></div></div>`;
          }
          break;

        case 'three-level':
          for (const [l1, l2Map] of grouped.tree) {
            const cnt = [...l2Map.values()].reduce((s, set) => s + set.size, 0);
            html += `<div class="rf-expansion-tree-group">`;
            html += `<div class="rf-expansion-tree-group-header" aria-expanded="true" onclick="toggleTreeGroup(this)">`;
            html += `<span class="toggle-arrow">\u25BC</span>`;
            html += `<span>${escapeHTML(l1)}</span>`;
            html += `<span class="group-count">(${cnt})</span></div>`;
            html += `<div class="rf-expansion-tree-items">`;
            for (const [l2, verbSet] of l2Map) {
              const verbs = [...verbSet].sort();
              html += `<div style="margin-bottom:4px;">`;
              html += `<span style="font-size:0.72rem;font-weight:600;color:#64748b;">${escapeHTML(l2)}</span> `;
              html += `<span class="rf-modal-expanded-pills" style="display:inline-flex;">`;
              for (const v of verbs) {
                html += `<span class="rf-modal-expanded-pill"><span class="pill-verb">${escapeHTML(v)}</span></span>`;
              }
              html += `</span></div>`;
            }
            html += `</div></div>`;
          }
          break;
      }
      return html;
    }

    function toggleTreeGroup(headerEl) {
      const items = headerEl.nextElementSibling;
      const arrow = headerEl.querySelector('.toggle-arrow');
      items.classList.toggle('collapsed');
      arrow.classList.toggle('collapsed');
      const isExpanded = !items.classList.contains('collapsed');
      headerEl.setAttribute('aria-expanded', String(isExpanded));
    }

    // -- Render rule cards (left column) -------------------------------------
    function renderSelectableRule(ref, index, color, selected) {
      const group = escapeHTML(formatGroupLabel(ref.apiGroup));
      const resource = escapeHTML(fmtResource(ref));
      const verb = escapeHTML(ref.verb || '*');
      const selCls = selected ? ' rf-rule-selected' : '';
      const unsupCls = ref.unsupportedVerb ? ' rf-rule-unsupported' : '';
      let html = `<div class="rf-modal-rule rf-rule-selectable${selCls}${unsupCls}" data-rule-idx="${index}" style="border-left:4px solid ${color}" onclick="onRuleToggle(${index})" onmouseenter="highlightExpansion(${index})" onmouseleave="unhighlightExpansion(${index})">`;
      html += `<div class="rf-rule-checkbox">`;
      html += `<input type="checkbox" ${selected ? 'checked' : ''} aria-label="Select rule: ${verb} ${group}/${resource}" onchange="onRuleToggle(${index})" onclick="event.stopPropagation()">`;
      html += `<div style="flex:1"><div class="rf-modal-rule-top">`;
      html += `<span class="rf-perm-verb">${verb}</span>`;
      html += `<span class="rf-perm-resource">${group}/${resource}</span>`;
      if (isWildcard(ref.apiGroup)) html += ' <span class="rf-modal-wildcard">* group</span>';
      if (isWildcard(ref.resource)) html += ' <span class="rf-modal-wildcard">* resource</span>';
      if (isWildcard(ref.verb)) html += ' <span class="rf-modal-wildcard">* verb</span>';
      if (ref.phantom) html += ' <span class="rf-phantom-tag">phantom</span>';
      if (ref.unsupportedVerb) html += ' <span class="rf-unsupported-badge">unsupported by API</span>';
      html += '</div>';
      if (ref.resourceNames && ref.resourceNames.length > 0) {
        html += `<div class="rf-perm-meta">names: ${ref.resourceNames.map(escapeHTML).join(', ')}</div>`;
      }
      if (Array.isArray(ref.expandedRefs) && ref.expandedRefs.length > 0) {
        html += `<div style="font-size:0.7rem;color:var(--muted);margin-top:2px;">${ref.expandedRefs.length} expanded permissions</div>`;
      }
      html += `</div></div></div>`;
      return html;
    }

    function renderConcreteRule(ref) {
      const group = escapeHTML(formatGroupLabel(ref.apiGroup));
      const resource = escapeHTML(fmtResource(ref));
      const verb = escapeHTML(ref.verb || '*');
      const unsupported = ref.unsupportedVerb === true;
      const cls = 'rf-modal-rule rf-rule-concrete' + (unsupported ? ' rf-rule-unsupported' : '');
      let html = `<div class="${cls}"><div class="rf-modal-rule-top">`;
      html += `<span class="rf-perm-verb">${verb}</span>`;
      html += `<span class="rf-perm-resource">${group}/${resource}</span>`;
      if (ref.phantom) html += ' <span class="rf-phantom-tag">phantom</span>';
      if (unsupported) html += ' <span class="rf-unsupported-badge">unsupported by API</span>';
      html += '</div>';
      if (ref.resourceNames && ref.resourceNames.length > 0) {
        html += `<div class="rf-perm-meta">names: ${ref.resourceNames.map(escapeHTML).join(', ')}</div>`;
      }
      html += '</div>';
      return html;
    }

    // -- Right column update -------------------------------------------------
    function updateExpansionColumn() {
      let html = '<h4>Expanded Permissions</h4>';

      if (modalWildcardRefs.length === 0) {
        html += '<div class="rf-modal-no-expansion">No wildcard rules to expand. All rules use concrete API groups, resources, and verbs.</div>';
        roleModalExpandedEl.innerHTML = html;
        return;
      }

      if (modalSelectedIndices.size === 0) {
        html += '<div class="rf-modal-no-expansion">Select a wildcard rule on the left to see its expansion.</div>';
        roleModalExpandedEl.innerHTML = html;
        return;
      }

      for (const idx of [...modalSelectedIndices].sort((a, b) => a - b)) {
        const ref = modalWildcardRefs[idx];
        if (!ref || !Array.isArray(ref.expandedRefs) || ref.expandedRefs.length === 0) continue;

        const color = RULE_COLORS[idx % RULE_COLORS.length];
        const wildcardType = classifyWildcardType(ref);
        const verb = escapeHTML(ref.verb || '*');
        const group = escapeHTML(formatGroupLabel(ref.apiGroup));
        const resource = escapeHTML(fmtResource(ref));

        html += `<div class="rf-expansion-section" data-rule-idx="${idx}" style="border-left:4px solid ${color}">`;
        html += `<div class="rf-expansion-header">`;
        html += `<span class="rule-summary"><span class="rf-perm-verb">${verb}</span> ${group}/${resource}</span>`;
        html += `<span class="rf-expansion-type-badge">${escapeHTML(wildcardTypeBadgeText(wildcardType))}</span>`;
        html += `<span class="rf-expansion-count-badge">${ref.expandedRefs.length} permission${ref.expandedRefs.length !== 1 ? 's' : ''}</span>`;
        html += `</div><div class="rf-expansion-tree">`;

        const grouped = groupExpandedRefs(ref.expandedRefs, wildcardType);
        html += renderGroupedExpansion(grouped);

        html += `</div></div>`;
      }

      roleModalExpandedEl.innerHTML = html;
    }

    // -- Event handlers ------------------------------------------------------
    function onRuleToggle(index) {
      if (modalSelectedIndices.has(index)) {
        modalSelectedIndices.delete(index);
      } else {
        modalSelectedIndices.add(index);
      }
      const card = roleModalOriginalEl.querySelector(`[data-rule-idx="${index}"]`);
      if (card) {
        card.classList.toggle('rf-rule-selected', modalSelectedIndices.has(index));
        const cb = card.querySelector('input[type="checkbox"]');
        if (cb) cb.checked = modalSelectedIndices.has(index);
      }
      updateExpansionColumn();
    }

    function selectAllRules(selectAll) {
      modalSelectedIndices = selectAll
        ? new Set(modalWildcardRefs.map((_, i) => i))
        : new Set();
      for (let i = 0; i < modalWildcardRefs.length; i++) {
        const card = roleModalOriginalEl.querySelector(`[data-rule-idx="${i}"]`);
        if (card) {
          card.classList.toggle('rf-rule-selected', selectAll);
          const cb = card.querySelector('input[type="checkbox"]');
          if (cb) cb.checked = selectAll;
        }
      }
      updateExpansionColumn();
    }

    function highlightExpansion(index) {
      const section = roleModalExpandedEl.querySelector(`.rf-expansion-section[data-rule-idx="${index}"]`);
      if (section) {
        section.classList.remove('highlight-pulse');
        void section.offsetWidth; // force reflow to restart animation
        section.classList.add('highlight-pulse');
        section.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
    }

    function unhighlightExpansion(index) {
      const section = roleModalExpandedEl.querySelector(`.rf-expansion-section[data-rule-idx="${index}"]`);
      if (section) section.classList.remove('highlight-pulse');
    }

    // -- Open / close modal --------------------------------------------------
    function openRoleModal(nodeId) {
      if (!lastGraph || !lastGraph.nodes) return;
      const node = lastGraph.nodes.find(n => n.id === nodeId);
      if (!node) return;
      const refs = Array.isArray(node.matchedRuleRefs) ? node.matchedRuleRefs : [];
      if (refs.length === 0) return;

      roleModalTitleEl.textContent = (node.type === 'role' || node.type === 'clusterRole')
        ? `${node.type}: ${node.name}`
        : node.name || nodeId;

      // Separate wildcard and concrete refs
      modalWildcardRefs = [];
      modalConcreteRefs = [];
      for (const ref of refs) {
        if (isWildcard(ref.apiGroup) || isWildcard(ref.resource) || isWildcard(ref.verb)) {
          modalWildcardRefs.push(ref);
        } else {
          modalConcreteRefs.push(ref);
        }
      }

      // Select all wildcard refs by default
      modalSelectedIndices = new Set(modalWildcardRefs.map((_, i) => i));

      // Build left column
      let leftHTML = '<h4>Original Rules</h4>';

      if (modalWildcardRefs.length > 1) {
        leftHTML += `<div class="rf-select-controls">`;
        leftHTML += `<button onclick="selectAllRules(true)">Select All</button>`;
        leftHTML += `<button onclick="selectAllRules(false)">Deselect All</button>`;
        leftHTML += `<span class="rf-select-label">${modalWildcardRefs.length} wildcard rules</span>`;
        leftHTML += `</div>`;
      }

      for (let i = 0; i < modalWildcardRefs.length; i++) {
        const color = RULE_COLORS[i % RULE_COLORS.length];
        leftHTML += renderSelectableRule(modalWildcardRefs[i], i, color, true);
      }

      if (modalConcreteRefs.length > 0) {
        if (modalWildcardRefs.length > 0) {
          leftHTML += `<div style="border-top:1px solid var(--line);margin:10px 0;padding-top:4px;font-size:0.7rem;color:var(--muted);text-transform:uppercase;letter-spacing:0.05em;">Concrete Rules</div>`;
        }
        for (const ref of modalConcreteRefs) {
          leftHTML += renderConcreteRule(ref);
        }
      }

      roleModalOriginalEl.innerHTML = leftHTML;

      // Build right column
      updateExpansionColumn();

      roleModalEl.style.display = 'flex';
    }

    function closeRoleModal() {
      roleModalEl.style.display = 'none';
    }

    document.addEventListener('keydown', e => {
      if (e.key === 'Escape' && roleModalEl.style.display !== 'none') {
        closeRoleModal();
      }
    });
    // -- End Role Modal -------------------------------------------------------

    function csv(id) {
      const raw = document.getElementById(id).value.trim();
      if (!raw) return [];
      return raw.split(',').map(v => v.trim()).filter(v => v.length > 0);
    }

    function intOrDefault(value, fallback) {
      const parsed = Number.parseInt(String(value || '').trim(), 10);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        return fallback;
      }
      return parsed;
    }

    function uniqStrings(values) {
      const out = [];
      const seen = new Set();
      for (const value of values || []) {
        const token = String(value || '').trim();
        if (!token) continue;
        const key = token.toLowerCase();
        if (seen.has(key)) continue;
        seen.add(key);
        out.push(token);
      }
      return out;
    }

    function sortOptions(kind, values) {
      const list = uniqStrings(values);
      if (kind === 'verbs') {
        const order = ['get', 'list', 'watch', 'create', 'update', 'patch', 'delete', 'deletecollection', 'bind', 'escalate', 'impersonate', '*'];
        const orderIdx = new Map(order.map((v, idx) => [v, idx]));
        list.sort((a, b) => {
          const la = a.toLowerCase();
          const lb = b.toLowerCase();
          const ia = orderIdx.has(la) ? orderIdx.get(la) : Number.MAX_SAFE_INTEGER;
          const ib = orderIdx.has(lb) ? orderIdx.get(lb) : Number.MAX_SAFE_INTEGER;
          if (ia !== ib) return ia - ib;
          return la.localeCompare(lb);
        });
        return list;
      }
      if (kind === 'apiGroups') {
        list.sort((a, b) => {
          if (a === '<core>') return -1;
          if (b === '<core>') return 1;
          if (a === '*') return -1;
          if (b === '*') return 1;
          return a.localeCompare(b);
        });
        return list;
      }
      return list.sort((a, b) => a.localeCompare(b));
    }

    function selectedSelectorValues(kind) {
      const manualValues = csv(SELECTOR_INPUT_IDS[kind]);
      const checkedValues = Array.from(selectorOptionSelection[kind] || []);
      const values = sortOptions(kind, manualValues.concat(checkedValues));
      if (kind === 'apiGroups') {
        return values.map(value => (value === '<core>' ? '' : value));
      }
      return values;
    }

    function setSelectorDiscoverStatus(message, isError = false) {
      selectorDiscoverStatusEl.textContent = message;
      selectorDiscoverStatusEl.classList.toggle('error', !!isError);
    }

    function renderSelectorOptions(kind) {
      const listEl = SELECTOR_LIST_ELS[kind];
      const countEl = SELECTOR_COUNT_ELS[kind];
      const filter = (selectorOptionFilters[kind] || '').toLowerCase();
      const selected = selectorOptionSelection[kind];
      const allValues = selectorOptionCatalog[kind] || [];
      const displayLabel = kind === 'resources' ? (v => resourceDisplayLabels.get(v) || v) : (v => v);
      const visibleValues = allValues.filter(value => displayLabel(value).toLowerCase().includes(filter));

      listEl.innerHTML = '';
      if (visibleValues.length === 0) {
        const empty = document.createElement('div');
        empty.className = 'selector-empty';
        empty.textContent = allValues.length === 0 ? 'No discovered values yet' : 'No values match the filter';
        listEl.appendChild(empty);
      } else {
        for (const value of visibleValues) {
          const label = document.createElement('label');
          label.className = 'selector-item';
          const input = document.createElement('input');
          input.type = 'checkbox';
          input.value = value;
          input.checked = selected.has(value);
          input.dataset.kind = kind;
          const text = document.createElement('span');
          text.textContent = displayLabel(value);
          label.append(input, text);
          listEl.appendChild(label);
        }
      }

      countEl.textContent = `${selected.size} / ${allValues.length}`;
    }

    function updateSelectorSummary() {
      selectorSummaryEl.textContent = `selected: verbs=${selectorOptionSelection.verbs.size}, apiGroups=${selectorOptionSelection.apiGroups.size}, resources=${selectorOptionSelection.resources.size}, nonResourceURLs=${selectorOptionSelection.nonResourceURLs.size}`;
    }

    function collectSelectorOptionsFromResourceMap(rows) {
      const options = {
        verbs: [],
        apiGroups: [],
        resources: [],
        nonResourceURLs: []
      };

      for (const row of rows || []) {
        const verb = String(row?.verb || '').trim();
        if (verb) options.verbs.push(verb);

        const apiGroup = String(row?.apiGroup || '').trim();
        options.apiGroups.push(apiGroup === '' ? '<core>' : apiGroup);

        const resource = String(row?.resource || '').trim();
        if (!resource) continue;
        if (resource.startsWith('/')) {
          for (const url of resource.split(',').map(v => v.trim()).filter(Boolean)) {
            options.nonResourceURLs.push(url);
          }
          continue;
        }
        options.resources.push(resource);
        if (!resourceDisplayLabels.has(resource)) {
          resourceDisplayLabels.set(resource, formatResource(apiGroup, resource));
        }
      }

      return {
        verbs: sortOptions('verbs', options.verbs),
        apiGroups: sortOptions('apiGroups', options.apiGroups),
        resources: sortOptions('resources', options.resources),
        nonResourceURLs: sortOptions('nonResourceURLs', options.nonResourceURLs)
      };
    }

    function applySelectorOptions(options, replace) {
      if (replace) resourceDisplayLabels.clear();
      for (const kind of SELECTOR_KINDS) {
        const incoming = Array.isArray(options?.[kind]) ? options[kind] : [];
        selectorOptionCatalog[kind] = replace
          ? sortOptions(kind, incoming)
          : sortOptions(kind, selectorOptionCatalog[kind].concat(incoming));

        const available = new Set(selectorOptionCatalog[kind]);
        selectorOptionSelection[kind] = new Set(
          Array.from(selectorOptionSelection[kind]).filter(value => available.has(value))
        );
        renderSelectorOptions(kind);
      }
      updateSelectorSummary();
      rawStore.request = JSON.stringify(payload());
      updateRequestText();
    }

    function payload() {
      const namespaceScopeNamespaces = csv('namespaceScopeNamespaces');
      const namespaceScopeStrict = !!namespaceScopeStrictEl.checked;
      const spec = {
        selector: {
          apiGroups: selectedSelectorValues('apiGroups'),
          resources: selectedSelectorValues('resources'),
          verbs: selectedSelectorValues('verbs'),
          resourceNames: csv('resourceNames'),
          nonResourceURLs: selectedSelectorValues('nonResourceURLs')
        },
        matchMode: document.getElementById('matchMode').value,
        wildcardMode: document.getElementById('wildcardMode').value,
        includeRuleMetadata: true,
        includePods: !!includePodsEl.checked,
        includeWorkloads: !!includeWorkloadsEl.checked,
        filterPhantomAPIs: !!filterPhantomAPIsEl.checked,
        podPhaseMode: podPhaseModeEl.value,
        maxPodsPerSubject: intOrDefault(maxPodsPerSubjectEl.value, 20),
        maxWorkloadsPerPod: intOrDefault(maxWorkloadsPerPodEl.value, 10)
      };
      if (namespaceScopeNamespaces.length > 0 || namespaceScopeStrict) {
        spec.namespaceScope = {
          namespaces: namespaceScopeNamespaces,
          strict: namespaceScopeStrict
        };
      }
      return {
        spec: spec
      };
    }

    function parseJSON(value) {
      try {
        return JSON.parse(value);
      } catch {
        return null;
      }
    }

    function formatByMode(raw, mode) {
      if (!raw) return '';
      if (mode !== 'pretty') return raw;
      const parsed = parseJSON(raw);
      if (!parsed) return raw;
      return JSON.stringify(parsed, null, 2);
    }

    function updateRequestText() {
      rawRequestEl.value = formatByMode(rawStore.request, requestViewEl.value);
    }

    function updateResponseText() {
      rawResponseEl.value = formatByMode(rawStore.response, responseViewEl.value);
    }

    async function copyFromElement(el, btn) {
      const text = el.value || '';
      if (!text) return;
      const previous = btn.textContent;
      try {
        await navigator.clipboard.writeText(text);
        btn.textContent = 'Copied';
      } catch {
        btn.textContent = 'Copy failed';
      }
      setTimeout(() => { btn.textContent = previous; }, 1200);
    }

    function isRoleNodeType(type) {
      return type === 'role' || type === 'clusterRole';
    }

    function nodeLayer(type) {
      if (type === 'role' || type === 'clusterRole') return 'Roles';
      if (type === 'roleBinding' || type === 'clusterRoleBinding') return 'Bindings';
      if (type === 'aggregationRelation') return 'Aggregation';
      if (type === 'permission') return 'Permissions';
      if (type === 'pod' || type === 'podOverflow') return 'Pods';
      if (type === 'workload' || type === 'workloadOverflow') return 'Workloads';
      return 'Subjects';
    }

    function normalizeRoleRefId(id) {
      if (!id) return id;
      if (id.startsWith('role:clusterrole:') || id.startsWith('role:role:')) {
        return id;
      }
      if (id.startsWith('clusterrole:')) {
        return `role:${id}`;
      }
      return id;
    }

    function roleNameFromId(roleId, nodeByID) {
      const node = nodeByID.get(roleId);
      if (node && node.name) return node.name;
      return roleId
        .replace(/^role:clusterrole:/, '')
        .replace(/^role:role:/, '');
    }

    function currentGraphOptions() {
      const runtimeView = runtimeViewEl?.value === 'ownership' ? 'ownership' : 'access';
      return {
        includeAggregates: !!showAggregateEdgesEl.checked,
        onlyReachable: !!onlyReachableEl.checked,
        showRolePermissions: !!showRolePermissionsEl.checked,
        showPermissions: !!showPermissionsEl.checked,
        spreadEdges: !!spreadEdgesEl.checked,
        focusMode: !!focusModeEl.checked,
        runtimeView,
        laneSpacing: Number.parseInt(String(laneSpacingEl?.value || ''), 10) || 540,
        rowSpacing: Number.parseInt(String(rowSpacingEl?.value || ''), 10) || 52
      };
    }

    function fanoutBucket(index, total) {
      const maxBucket = EDGE_HANDLE_POSITIONS.length - 1;
      if (maxBucket <= 0 || total <= 1) {
        return Math.max(0, Math.floor(maxBucket / 2));
      }
      const ratio = index / (total - 1);
      return Math.max(0, Math.min(maxBucket, Math.round(ratio * maxBucket)));
    }

    function edgeFamily(edge) {
      const type = String(edge?.type || '');
      if (type.startsWith('aggregates')) return 'aggregate';
      if (type.startsWith('permissions-')) return 'permission';
      if (type === 'ownedBy') return 'ownership';
      if (type === 'runsAs') return 'runtime';
      return 'structural';
    }

    function handleBandForFamily(family) {
      return EDGE_BANDS[family] || EDGE_BANDS.structural;
    }

    function pickBandIndex(index, total, band) {
      if (!Array.isArray(band) || band.length === 0) return fanoutBucket(index, total);
      if (band.length === 1 || total <= 1) return band[Math.floor((band.length - 1) / 2)];
      const ratio = index / (total - 1);
      const slot = Math.max(0, Math.min(band.length - 1, Math.round(ratio * (band.length - 1))));
      return band[slot];
    }

    function buildHandleBandBucketMap(edges, keyField, peerField, nodeYByID) {
      const grouped = new Map();
      for (const edge of edges) {
        const nodeKey = String(edge[keyField] || '');
        const family = edgeFamily(edge);
        const groupKey = `${nodeKey}::${family}`;
        if (!grouped.has(groupKey)) grouped.set(groupKey, []);
        grouped.get(groupKey).push(edge);
      }

      const buckets = new Map();
      for (const list of grouped.values()) {
        list.sort((a, b) => {
          const ay = Number(nodeYByID.get(a[peerField]) ?? 0);
          const by = Number(nodeYByID.get(b[peerField]) ?? 0);
          if (ay !== by) return ay - by;
          const aKey = `${a.from}|${a.to}|${a.type}|${a.id}`;
          const bKey = `${b.from}|${b.to}|${b.type}|${b.id}`;
          return aKey.localeCompare(bKey);
        });
        const family = edgeFamily(list[0]);
        const band = handleBandForFamily(family);
        const total = list.length;
        list.forEach((edge, idx) => {
          buckets.set(edge.id, pickBandIndex(idx, total, band));
        });
      }
      return buckets;
    }

    function estimateCardHeight(node, sourceNames, hiddenSources, inlinePermissionPreviewCount, inlinePermissionHiddenCount) {
      const title = node.name || node.id || '';
      const objectID = node.id || '';
      const titleLines = Math.max(1, Math.ceil(title.length / 24));
      const idLines = Math.max(1, Math.ceil(objectID.length / 42));
      let height = CARD_BASE_HEIGHT + (titleLines * CARD_TITLE_LINE_HEIGHT) + (idLines * CARD_ID_LINE_HEIGHT);
      if (sourceNames.length > 0) {
        height += 34 + (sourceNames.length * CARD_TITLE_LINE_HEIGHT);
        if (hiddenSources > 0) height += CARD_TITLE_LINE_HEIGHT;
      }
      if (inlinePermissionPreviewCount > 0) {
        height += 34 + (inlinePermissionPreviewCount * 24);
        if (inlinePermissionHiddenCount > 0) height += CARD_TITLE_LINE_HEIGHT;
      }
      return Math.max(height, CARD_MIN_HEIGHT);
    }

    function permissionKeyFromRef(ref) {
      const isNonResource = Array.isArray(ref.nonResourceURLs) && ref.nonResourceURLs.length > 0;
      const apiGroup = isNonResource
        ? '<nonResource>'
        : ((ref.apiGroup === undefined || ref.apiGroup === null || ref.apiGroup === '') ? '<core>' : String(ref.apiGroup));
      const resourceValue = isNonResource
        ? ref.nonResourceURLs.join(', ')
        : ((ref.resource || '*') + (ref.subresource ? `/${ref.subresource}` : ''));
      const verb = ref.verb || '*';
      return {
        id: `permission:${apiGroup}|${resourceValue}|${verb}`,
        apiGroup,
        resourceValue,
        verb,
        isNonResource,
        phantom: !!ref.phantom
      };
    }

    function formatResource(apiGroup, resourceValue) {
      const group = (!apiGroup || apiGroup === '<core>') ? 'core' : apiGroup;
      return group + '/' + (resourceValue || '*');
    }

    function comparePermissionNodes(a, b) {
      if ((a.permissionAPIGroup || '') !== (b.permissionAPIGroup || '')) {
        return (a.permissionAPIGroup || '').localeCompare(b.permissionAPIGroup || '');
      }
      if ((a.name || '') !== (b.name || '')) {
        return (a.name || '').localeCompare(b.name || '');
      }
      return (a.permissionVerb || '').localeCompare(b.permissionVerb || '');
    }

    function buildPermissionProjection(nodes, edges, nodeByID) {
      const permissionNodeByID = new Map();
      const permissionEdges = [];
      const edgeSeen = new Set();
      const roleWithBindings = new Set();

      function ensurePermissionNode(ref) {
        const key = permissionKeyFromRef(ref);
        if (!permissionNodeByID.has(key.id)) {
          permissionNodeByID.set(key.id, {
            id: key.id,
            type: 'permission',
            name: key.resourceValue,
            permissionVerb: key.verb,
            permissionAPIGroup: key.apiGroup,
            permissionNonResource: key.isNonResource,
            phantom: key.phantom
          });
        }
        return key.id;
      }

      function addPermissionEdge(from, to, type, explain) {
        if (!nodeByID.has(from) && !permissionNodeByID.has(from)) return;
        if (!nodeByID.has(to) && !permissionNodeByID.has(to)) return;
        const edgeID = `${from}->${to}:${type}`;
        if (edgeSeen.has(edgeID)) return;
        edgeSeen.add(edgeID);
        permissionEdges.push({
          id: `edge:${edgeID}`,
          from,
          to,
          type,
          explain
        });
      }

      for (const edge of edges) {
        if (edge.type !== 'grants') continue;
        if (!nodeByID.has(edge.from) || !nodeByID.has(edge.to)) continue;
        roleWithBindings.add(edge.from);
        const refs = Array.isArray(edge.ruleRefs) ? edge.ruleRefs : [];
        for (const ref of refs) {
          const permissionID = ensurePermissionNode(ref);
          addPermissionEdge(edge.from, permissionID, 'permissions-role', 'Role contains selected permission');
          addPermissionEdge(permissionID, edge.to, 'permissions-binding', 'Binding references role with selected permission');
        }
      }

      // Connect unbound matched roles directly to permissions so aggregated-only chains stay visible.
      for (const node of nodes) {
        if (!isRoleNodeType(node.type)) continue;
        if (roleWithBindings.has(node.id)) continue;
        const refs = Array.isArray(node.matchedRuleRefs) ? node.matchedRuleRefs : [];
        for (const ref of refs) {
          const permissionID = ensurePermissionNode(ref);
          addPermissionEdge(node.id, permissionID, 'permissions-role', 'Role matches selector (no direct binding in result)');
        }
      }

      const permissionNodes = Array.from(permissionNodeByID.values()).sort(comparePermissionNodes);
      return { permissionNodes, permissionEdges };
    }

    function buildRolePermissionMap(nodes) {
      const rolePermissionsByRoleID = new Map();
      for (const node of nodes || []) {
        if (!isRoleNodeType(node?.type)) continue;
        const refs = Array.isArray(node.matchedRuleRefs) ? node.matchedRuleRefs : [];
        if (refs.length === 0) continue;
        const permissionByID = new Map();
        for (const ref of refs) {
          const key = permissionKeyFromRef(ref);
          if (!permissionByID.has(key.id)) {
            permissionByID.set(key.id, {
              id: key.id,
              name: key.resourceValue,
              permissionVerb: key.verb,
              permissionAPIGroup: key.apiGroup,
              permissionNonResource: key.isNonResource,
              phantom: key.phantom
            });
          }
        }
        rolePermissionsByRoleID.set(node.id, Array.from(permissionByID.values()).sort(comparePermissionNodes));
      }
      return rolePermissionsByRoleID;
    }

    function sortByIncomingParentOrder(nodes, incomingEdges, parentOrderMap, tieBreaker) {
      const scoreByID = new Map();
      for (const node of nodes) {
        let sum = 0;
        let count = 0;
        for (const edge of incomingEdges) {
          if (edge.to !== node.id) continue;
          const parentOrder = parentOrderMap.get(edge.from);
          if (parentOrder === undefined) continue;
          sum += parentOrder;
          count += 1;
        }
        scoreByID.set(node.id, count > 0 ? (sum / count) : Number.POSITIVE_INFINITY);
      }

      nodes.sort((a, b) => {
        const scoreA = scoreByID.get(a.id);
        const scoreB = scoreByID.get(b.id);
        if (scoreA !== scoreB) return scoreA - scoreB;
        const ta = (tieBreaker(a) || '').toLowerCase();
        const tb = (tieBreaker(b) || '').toLowerCase();
        return ta.localeCompare(tb);
      });
    }

    function applyFocusToModel(baseModel, focusNodeID, focusModeEnabled) {
      const nodes = Array.isArray(baseModel?.nodes) ? baseModel.nodes : [];
      const edges = Array.isArray(baseModel?.edges) ? baseModel.edges : [];
      if (!focusModeEnabled || !focusNodeID || !nodes.some(node => node.id === focusNodeID)) {
        return {
          nodes: nodes.map(node => ({
            ...node,
            data: { ...(node.data || {}), focusDim: false, focusRoot: false }
          })),
          edges: edges.map(edge => ({
            ...edge,
            style: { ...(edge.style || {}) },
            animated: !!edge.animated
          }))
        };
      }

      const outgoing = new Map();
      const incoming = new Map();
      for (const edge of edges) {
        if (!outgoing.has(edge.source)) outgoing.set(edge.source, []);
        outgoing.get(edge.source).push(edge);
        if (!incoming.has(edge.target)) incoming.set(edge.target, []);
        incoming.get(edge.target).push(edge);
      }

      const activeNodes = new Set([focusNodeID]);
      const activeEdges = new Set();

      const qOut = [focusNodeID];
      while (qOut.length > 0) {
        const nodeID = qOut.shift();
        const outs = outgoing.get(nodeID) || [];
        for (const edge of outs) {
          activeEdges.add(edge.id);
          if (!activeNodes.has(edge.target)) {
            activeNodes.add(edge.target);
            qOut.push(edge.target);
          }
        }
      }

      const qIn = [focusNodeID];
      while (qIn.length > 0) {
        const nodeID = qIn.shift();
        const ins = incoming.get(nodeID) || [];
        for (const edge of ins) {
          activeEdges.add(edge.id);
          if (!activeNodes.has(edge.source)) {
            activeNodes.add(edge.source);
            qIn.push(edge.source);
          }
        }
      }

      return {
        nodes: nodes.map(node => {
          const active = activeNodes.has(node.id);
          return {
            ...node,
            data: {
              ...(node.data || {}),
              focusDim: !active,
              focusRoot: node.id === focusNodeID
            }
          };
        }),
        edges: edges.map(edge => {
          const active = activeEdges.has(edge.id);
          const baseOpacity = Number(edge.style?.opacity ?? 0.9);
          const baseWidth = Number(edge.style?.strokeWidth ?? 1.8);
          return {
            ...edge,
            animated: active ? !!edge.animated : false,
            style: {
              ...(edge.style || {}),
              opacity: active ? baseOpacity : 0.08,
              strokeWidth: active ? baseWidth : Math.max(1, baseWidth - 0.6)
            },
            zIndex: active ? 10 : 1
          };
        })
      };
    }

    function runtimeDisplayEdges(edges, runtimeView) {
      if (!Array.isArray(edges) || edges.length === 0) return [];
      if (runtimeView !== 'ownership') {
        return edges.slice();
      }
      return edges.map(edge => {
        if (edge.type !== 'ownedBy') {
          return edge;
        }
        return {
          ...edge,
          from: edge.to,
          to: edge.from,
          explain: edge.explain || 'Ownership path shown as owner -> dependent'
        };
      });
    }

    function updateRuntimeLegend() {
      const mode = runtimeViewEl?.value === 'ownership' ? 'ownership' : 'access';
      if (legendRunsAsTextEl) {
        legendRunsAsTextEl.textContent = 'subject -> pod (runsAs)';
      }
      if (legendOwnedByTextEl) {
        legendOwnedByTextEl.textContent = mode === 'ownership'
          ? 'owner workload -> dependent workload/pod (ownedBy)'
          : 'pod -> workload owner chain (ownedBy)';
      }
    }

    function fitFlowInstance(instance, duration) {
      if (!instance) return;
      instance.fitView({
        padding: 0.1,
        duration: Number.isFinite(duration) ? duration : 220,
        includeHiddenNodes: false,
        maxZoom: 1.05
      });
    }

    function buildFlowModel(graph, options) {
      const includeAggregates = !!options?.includeAggregates;
      const onlyReachable = !!options?.onlyReachable;
      const showRolePermissions = !!options?.showRolePermissions;
      const showPermissions = !!options?.showPermissions;
      const spreadEdges = !!options?.spreadEdges;
      const runtimeView = options?.runtimeView === 'ownership' ? 'ownership' : 'access';
      const allNodes = graph?.nodes || [];
      const allRawEdges = graph?.edges || [];

      const structuralEdges = allRawEdges.filter(edge => edge.type !== 'aggregates');
      let nodeIDsToRender = null;
      if (onlyReachable) {
        nodeIDsToRender = new Set();
        for (const edge of structuralEdges) {
          nodeIDsToRender.add(edge.from);
          nodeIDsToRender.add(edge.to);
        }
      }

      const visibleBaseNodes = nodeIDsToRender
        ? allNodes.filter(node => nodeIDsToRender.has(node.id))
        : allNodes.slice();
      const visibleBaseNodeIDSet = new Set(visibleBaseNodes.map(node => node.id));
      const visibleBaseNodeByID = new Map(visibleBaseNodes.map(node => [node.id, node]));
      const filteredDirectEdges = structuralEdges
        .filter(edge => visibleBaseNodeIDSet.has(edge.from) && visibleBaseNodeIDSet.has(edge.to));
      const rolePermissionsByRoleID = buildRolePermissionMap(visibleBaseNodes);

      const aggregatedTargetRoleIDs = new Set();
      for (const node of visibleBaseNodes) {
        if (!isRoleNodeType(node.type) || !node.aggregated || !Array.isArray(node.aggregationSources)) {
          continue;
        }
        aggregatedTargetRoleIDs.add(node.id);
      }

      const sourceLaneRoleIDs = new Set();
      const aggregationRelationNodes = [];
      const aggregationRelationEdges = [];
      if (includeAggregates) {
        for (const targetNode of visibleBaseNodes) {
          if (!isRoleNodeType(targetNode.type) || !targetNode.aggregated || !Array.isArray(targetNode.aggregationSources)) {
            continue;
          }

          const relationSources = [];
          for (const rawSourceID of targetNode.aggregationSources) {
            const sourceID = normalizeRoleRefId(rawSourceID);
            const sourceNode = visibleBaseNodeByID.get(sourceID);
            if (!sourceNode || !isRoleNodeType(sourceNode.type)) {
              continue;
            }
            // Keep direction consistent in the UI: source lane contains non-target contributors.
            if (aggregatedTargetRoleIDs.has(sourceID)) {
              continue;
            }
            relationSources.push(sourceID);
            sourceLaneRoleIDs.add(sourceID);
          }

          if (relationSources.length === 0) {
            continue;
          }

          const relationID = `aggregation:${targetNode.id}`;
          aggregationRelationNodes.push({
            id: relationID,
            type: 'aggregationRelation',
            name: `aggregate -> ${targetNode.name}`,
            relationTarget: targetNode.id,
            relationTargetName: targetNode.name,
            relationSourceCount: relationSources.length
          });

          for (const sourceID of relationSources) {
            aggregationRelationEdges.push({
              id: `edge:${sourceID}->${relationID}:aggregates-source`,
              from: sourceID,
              to: relationID,
              type: 'aggregates-source'
            });
          }

          aggregationRelationEdges.push({
            id: `edge:${relationID}->${targetNode.id}:aggregates-target`,
            from: relationID,
            to: targetNode.id,
            type: 'aggregates-target'
          });
        }
      }

      const permissionProjection = showPermissions
        ? buildPermissionProjection(visibleBaseNodes, filteredDirectEdges, visibleBaseNodeByID)
        : { permissionNodes: [], permissionEdges: [] };
      const directEdgesBeforeRuntimeView = showPermissions
        ? filteredDirectEdges.filter(edge => edge.type !== 'grants')
        : filteredDirectEdges;
      const directEdgesForRender = runtimeDisplayEdges(directEdgesBeforeRuntimeView, runtimeView);

      const visibleNodes = visibleBaseNodes
        .concat(aggregationRelationNodes)
        .concat(permissionProjection.permissionNodes);
      const visibleNodeByID = new Map(visibleNodes.map(node => [node.id, node]));

      const ownershipEdges = directEdgesForRender.filter(edge => edge.type === 'ownedBy');
      const ownershipWorkloadIncoming = new Set();
      const ownershipWorkloadOutgoing = new Set();
      for (const edge of ownershipEdges) {
        const fromNode = visibleBaseNodeByID.get(edge.from);
        const toNode = visibleBaseNodeByID.get(edge.to);
        if (fromNode?.type === 'workload') ownershipWorkloadOutgoing.add(edge.from);
        if (toNode?.type === 'workload') ownershipWorkloadIncoming.add(edge.to);
      }
      const ownershipRootWorkloadIDs = new Set();
      for (const node of visibleBaseNodes) {
        if (node.type !== 'workload') continue;
        if (ownershipWorkloadOutgoing.has(node.id) && !ownershipWorkloadIncoming.has(node.id)) {
          ownershipRootWorkloadIDs.add(node.id);
        }
      }

      const hasAggSourcesLane = sourceLaneRoleIDs.size > 0;
      const hasAggregationLane = aggregationRelationNodes.length > 0;
      const hasAggregatedRolesLane = visibleBaseNodes.some(node => isRoleNodeType(node.type) && !!node.aggregated);
      const hasPermissionsLane = permissionProjection.permissionNodes.length > 0;
      const hasPodsLane = visibleBaseNodes.some(node => node.type === 'pod' || node.type === 'podOverflow');
      const hasWorkloadsLane = visibleBaseNodes.some(node => node.type === 'workload' || node.type === 'workloadOverflow');
      const hasWorkloadOwnersLane = runtimeView === 'ownership' && visibleBaseNodes.some(node => node.type === 'workload' && ownershipRootWorkloadIDs.has(node.id));
      const laneOrder = [
        'Roles',
        ...(hasAggSourcesLane ? ['AggSources'] : []),
        ...(hasAggregationLane ? ['Aggregation'] : []),
        ...(hasAggregatedRolesLane ? ['AggregatedRoles'] : []),
        ...(hasPermissionsLane ? ['Permissions'] : []),
        'Bindings',
        'Subjects',
        ...(runtimeView === 'ownership'
          ? [
              ...(hasWorkloadOwnersLane ? ['WorkloadOwners'] : []),
              ...(hasWorkloadsLane ? ['Workloads'] : []),
              ...(hasPodsLane ? ['Pods'] : []),
            ]
          : [
              ...(hasPodsLane ? ['Pods'] : []),
              ...(hasWorkloadsLane ? ['Workloads'] : []),
            ])
      ];
      const grouped = {
        AggSources: [],
        Aggregation: [],
        Roles: [],
        AggregatedRoles: [],
        Bindings: [],
        Permissions: [],
        Subjects: [],
        WorkloadOwners: [],
        Pods: [],
        Workloads: []
      };

      for (const node of visibleNodes) {
        if (node.type === 'aggregationRelation') {
          grouped.Aggregation.push(node);
          continue;
        }
        if (node.type === 'permission') {
          grouped.Permissions.push(node);
          continue;
        }
        const layer = nodeLayer(node.type);
        if (runtimeView === 'ownership' && layer === 'Workloads' && ownershipRootWorkloadIDs.has(node.id)) {
          grouped.WorkloadOwners.push(node);
          continue;
        }
        if (layer === 'Roles' && hasAggSourcesLane && sourceLaneRoleIDs.has(node.id)) {
          grouped.AggSources.push(node);
          continue;
        }
        if (layer === 'Roles' && hasAggregatedRolesLane && node.aggregated) {
          grouped.AggregatedRoles.push(node);
          continue;
        }
        grouped[layer].push(node);
      }
      for (const layer of Object.keys(grouped)) {
        grouped[layer].sort((a, b) => {
          if (layer === 'Aggregation') {
            return (a.relationTargetName || '').localeCompare(b.relationTargetName || '');
          }
          if (layer === 'Permissions') {
            return comparePermissionNodes(a, b);
          }
          const left = (a.name || a.id || '').toLowerCase();
          const right = (b.name || b.id || '').toLowerCase();
          return left.localeCompare(right);
        });
      }

      const aggregationTargetOrderMap = new Map();
      grouped.AggregatedRoles.forEach((node, idx) => aggregationTargetOrderMap.set(node.id, idx));
      grouped.Roles.forEach((node, idx) => {
        if (!aggregationTargetOrderMap.has(node.id)) {
          aggregationTargetOrderMap.set(node.id, grouped.AggregatedRoles.length + idx);
        }
      });
      if (grouped.Aggregation.length > 0) {
        grouped.Aggregation.sort((a, b) => {
          const oa = aggregationTargetOrderMap.get(a.relationTarget);
          const ob = aggregationTargetOrderMap.get(b.relationTarget);
          const va = (oa === undefined) ? Number.POSITIVE_INFINITY : oa;
          const vb = (ob === undefined) ? Number.POSITIVE_INFINITY : ob;
          if (va !== vb) return va - vb;
          return (a.relationTargetName || '').localeCompare(b.relationTargetName || '');
        });
      }

      const laneStartX = 24;
      const laneStartY = 24;
      const laneGapY = Number.isFinite(options?.rowSpacing) ? Math.max(24, options.rowSpacing) : 52;
      const cardWidth = CARD_WIDTH;
      const sourcePreviewLimit = SOURCE_PREVIEW_LIMIT;
      const permissionPreviewLimit = PERMISSION_PREVIEW_LIMIT;
      const configuredLaneGapX = Number.isFinite(options?.laneSpacing) ? options.laneSpacing : 540;
      const laneGapX = laneOrder.length <= 1
        ? 0
        : Math.max(360, configuredLaneGapX);
      const layerX = {};
      laneOrder.forEach((lane, idx) => {
        layerX[lane] = laneStartX + idx * laneGapX;
      });

      const permissionParentOrderMap = new Map();
      grouped.Roles.forEach((node, idx) => permissionParentOrderMap.set(node.id, idx));
      grouped.AggSources.forEach((node, idx) => permissionParentOrderMap.set(node.id, grouped.Roles.length + idx));
      grouped.AggregatedRoles.forEach((node, idx) => permissionParentOrderMap.set(node.id, grouped.Roles.length + grouped.AggSources.length + idx));
      if (grouped.Permissions.length > 0) {
        sortByIncomingParentOrder(
          grouped.Permissions,
          permissionProjection.permissionEdges.filter(edge => edge.type === 'permissions-role'),
          permissionParentOrderMap,
          node => `${node.permissionAPIGroup || ''}|${node.name || ''}|${node.permissionVerb || ''}`
        );
      }
      if (grouped.Bindings.length > 0 && grouped.Permissions.length > 0) {
        const permissionOrderMap = new Map(grouped.Permissions.map((node, idx) => [node.id, idx]));
        sortByIncomingParentOrder(
          grouped.Bindings,
          permissionProjection.permissionEdges.filter(edge => edge.type === 'permissions-binding'),
          permissionOrderMap,
          node => `${node.type || ''}|${node.name || ''}|${node.namespace || ''}`
        );
      }
      const bindingOrderMap = new Map(grouped.Bindings.map((node, idx) => [node.id, idx]));
      if (grouped.Subjects.length > 0) {
        sortByIncomingParentOrder(
          grouped.Subjects,
          filteredDirectEdges.filter(edge => edge.type === 'subjects'),
          bindingOrderMap,
          node => `${node.type || ''}|${node.name || ''}|${node.namespace || ''}`
        );
      }
      if (grouped.Pods.length > 0) {
        const subjectOrderMap = new Map(grouped.Subjects.map((node, idx) => [node.id, idx]));
        grouped.Workloads.forEach((node, idx) => {
          if (!subjectOrderMap.has(node.id)) {
            subjectOrderMap.set(node.id, grouped.Subjects.length + idx);
          }
        });
        const podIncomingEdges = directEdgesForRender.filter(edge => edge.type === 'runsAs' || edge.type === 'ownedBy');
        sortByIncomingParentOrder(
          grouped.Pods,
          podIncomingEdges,
          subjectOrderMap,
          node => `${node.namespace || ''}|${node.name || ''}|${node.id || ''}`
        );
      }
      if (grouped.Workloads.length > 0) {
        const workloadParentOrderMap = new Map();
        grouped.WorkloadOwners.forEach((node, idx) => workloadParentOrderMap.set(node.id, idx));
        grouped.Pods.forEach((node, idx) => workloadParentOrderMap.set(node.id, idx));
        grouped.Workloads.forEach((node, idx) => {
          if (!workloadParentOrderMap.has(node.id)) {
            workloadParentOrderMap.set(node.id, grouped.WorkloadOwners.length + grouped.Pods.length + idx);
          }
        });
        const workloadIncomingEdges = directEdgesForRender.filter(edge => edge.type === 'ownedBy');
        sortByIncomingParentOrder(
          grouped.Workloads,
          workloadIncomingEdges,
          workloadParentOrderMap,
          node => `${node.workloadKind || ''}|${node.namespace || ''}|${node.name || ''}|${node.id || ''}`
        );
      }

      const yPos = {};
      laneOrder.forEach(lane => { yPos[lane] = laneStartY; });
      const flowNodes = [];
      const existingIDs = new Set();

      const laneTitleByLayer = {
        AggSources: 'Aggregation Sources',
        Aggregation: 'Aggregation Links',
        AggregatedRoles: 'Aggregated Roles',
        WorkloadOwners: 'Workload Owners'
      };
      for (const layer of laneOrder) {
        for (const node of grouped[layer]) {
          const isPermissionCard = node.type === 'permission';
          const isAggregationRelationCard = node.type === 'aggregationRelation';
          const isPodCard = node.type === 'pod';
          const isWorkloadCard = node.type === 'workload';
          const isOverflowCard = node.type === 'podOverflow' || node.type === 'workloadOverflow';
          const inlineRolePermissions = (showRolePermissions && isRoleNodeType(node.type))
            ? (rolePermissionsByRoleID.get(node.id) || [])
            : [];
          const visibleInlineRolePermissions = inlineRolePermissions.slice(0, permissionPreviewLimit);
          const hiddenInlineRolePermissions = Math.max(0, inlineRolePermissions.length - visibleInlineRolePermissions.length);
          const sourceNames = isPermissionCard
            ? []
            : isAggregationRelationCard
              ? []
              : (node.aggregationSources || [])
              .map(normalizeRoleRefId)
              .filter(sourceID => visibleBaseNodeByID.has(sourceID))
              .map(sourceID => roleNameFromId(sourceID, visibleBaseNodeByID));
          const visibleSources = sourceNames.slice(0, sourcePreviewLimit);
          const hiddenSources = Math.max(0, sourceNames.length - visibleSources.length);
          const title = isPermissionCard
            ? (node.permissionNonResource ? (node.name || '*') : formatResource(node.permissionAPIGroup, node.name))
            : isAggregationRelationCard
              ? `aggregate -> ${node.relationTargetName || ''}`
              : (node.name || node.id);
          const subtype = isPermissionCard
            ? `permission | verb: ${node.permissionVerb}`
            : isAggregationRelationCard
              ? `aggregation relation | sources: ${node.relationSourceCount || 0}`
              : isPodCard
                ? `pod | phase: ${node.podPhase || '-'}${node.namespace ? ` | ns: ${node.namespace}` : ''}`
                : isWorkloadCard
                  ? `workload | kind: ${node.workloadKind || '-'}${node.namespace ? ` | ns: ${node.namespace}` : ''}`
                  : isOverflowCard
                    ? `${node.type} | hidden: ${node.hiddenCount || 0}${node.namespace ? ` | ns: ${node.namespace}` : ''}`
                    : (node.type + (node.namespace ? ` | ns: ${node.namespace}` : ''));
          const objectId = isPermissionCard
            ? `permission:${node.permissionAPIGroup}/${node.name}/${node.permissionVerb}`
            : isAggregationRelationCard
              ? `aggregation-target:${node.relationTarget || ''}`
              : node.id;
          const estHeight = (isPermissionCard || isPodCard || isWorkloadCard || isOverflowCard)
            ? 116
            : isAggregationRelationCard
              ? 92
              : estimateCardHeight(
                node,
                visibleSources,
                hiddenSources,
                visibleInlineRolePermissions.length,
                hiddenInlineRolePermissions
              );

          flowNodes.push({
            id: node.id,
            type: 'rbacCard',
            position: { x: layerX[layer], y: yPos[layer] },
            data: {
              layer,
              laneTitle: laneTitleByLayer[layer] || layer,
              sourceLane: layer === 'AggSources',
              permissionCard: isPermissionCard,
              aggregationRelationCard: isAggregationRelationCard,
              podCard: isPodCard,
              workloadCard: isWorkloadCard,
              overflowCard: isOverflowCard,
              title,
              subtype,
              objectId,
              aggregated: !isPermissionCard && !isAggregationRelationCard && !!node.aggregated,
              aggregationCount: sourceNames.length,
              aggregationSources: visibleSources,
              aggregationHidden: hiddenSources,
              inlineRolePermissionCount: inlineRolePermissions.length,
              inlineRolePermissions: visibleInlineRolePermissions,
              inlineRolePermissionHidden: hiddenInlineRolePermissions,
              phantom: !!node.phantom
            },
            draggable: false,
            selectable: true,
            style: { width: cardWidth }
          });

          yPos[layer] += estHeight + laneGapY;
          existingIDs.add(node.id);
        }
      }

      const combinedEdges = directEdgesForRender
        .concat(permissionProjection.permissionEdges)
        .concat(aggregationRelationEdges);
      const validCombinedEdges = combinedEdges.filter(edge => existingIDs.has(edge.from) && existingIDs.has(edge.to));
      const nodeYByID = new Map(flowNodes.map(node => [node.id, Number(node.position?.y) || 0]));
      const sourceBucketByEdgeID = spreadEdges
        ? buildHandleBandBucketMap(validCombinedEdges, 'from', 'to', nodeYByID)
        : new Map();
      const targetBucketByEdgeID = spreadEdges
        ? buildHandleBandBucketMap(validCombinedEdges, 'to', 'from', nodeYByID)
        : new Map();
      const flowEdges = [];
      let edgeIdx = 0;
      const markerType = window.ReactFlow?.MarkerType?.ArrowClosed || 'arrowclosed';
      for (const edge of validCombinedEdges) {
        const isAggregate = edge.type === 'aggregates-source' || edge.type === 'aggregates-target';
        const isGrant = edge.type === 'grants';
        const isRunsAs = edge.type === 'runsAs';
        const isOwnedBy = edge.type === 'ownedBy';
        const isPermissionEdge =
          edge.type === 'permissions-binding'
          || edge.type === 'permissions-role';
        const isPermissionFromRole = edge.type === 'permissions-role';
        const sourceNode = visibleNodeByID.get(edge.from);
        const targetNode = visibleNodeByID.get(edge.to);
        if (!sourceNode || !targetNode) continue;

        const color = isAggregate
          ? COLORS.edgeAggregates
          : (isPermissionEdge
            ? COLORS.edgePermissions
            : (isRunsAs
              ? COLORS.edgeRunsAs
              : (isOwnedBy
                ? COLORS.edgeOwnedBy
                : (isGrant ? COLORS.edgeGrants : COLORS.edgeBinds))));
        const edgeType = isOwnedBy
          ? (runtimeView === 'ownership' ? 'step' : 'smoothstep')
          : ((isAggregate || isPermissionEdge || isRunsAs) ? 'smoothstep' : 'bezier');
        const family = edgeFamily(edge);
        const band = handleBandForFamily(family);
        const centerHandleIdx = band[Math.floor((band.length - 1) / 2)];
        const sourceHandle = spreadEdges
          ? `out-right-${sourceBucketByEdgeID.get(edge.id) ?? centerHandleIdx}`
          : `out-right-${centerHandleIdx}`;
        const targetHandle = spreadEdges
          ? `in-left-${targetBucketByEdgeID.get(edge.id) ?? centerHandleIdx}`
          : `in-left-${centerHandleIdx}`;

        flowEdges.push({
          id: `edge-${edgeIdx++}-${edge.from}-${edge.to}-${edge.type}`,
          source: edge.from,
          target: edge.to,
          sourceHandle,
          targetHandle,
          type: edgeType,
          animated: isGrant || isPermissionFromRole,
          style: {
            stroke: color,
            strokeWidth: isAggregate ? 2 : (isPermissionEdge ? 2 : (isRunsAs || isOwnedBy ? 1.9 : (isGrant ? 2.1 : 1.8))),
            strokeDasharray: edge.type === 'aggregates-source' || isPermissionFromRole || (isRunsAs && runtimeView === 'ownership') ? '7 5' : undefined,
            opacity: isAggregate ? 0.72 : 0.9
          },
          pathOptions: {
            offset: 34,
            borderRadius: 10
          },
          markerEnd: {
            type: markerType,
            color,
            width: 18,
            height: 18
          }
        });
      }

      return { nodes: flowNodes, edges: flowEdges };
    }

    function mountFlowApp() {
      if (!window.React || !window.ReactDOM || !window.ReactFlow) {
        statusEl.classList.add('error');
        statusEl.textContent = 'React Flow assets failed to load';
        return;
      }

      const React = window.React;
      const ReactDOM = window.ReactDOM;
      const RF = window.ReactFlow;
      const h = React.createElement;

      function RBACNodeCard(props) {
        const data = props.data || {};
        const badges = [];
        if (data.sourceLane) {
          badges.push(h('span', { key: 'source-badge', className: 'rf-badge source' }, 'aggregation-source'));
        }
        if (data.aggregationRelationCard) {
          badges.push(h('span', { key: 'aggrel-badge', className: 'rf-badge aggregation-rel' }, 'aggregation-link'));
        }
        if (data.aggregated) {
          badges.push(h('span', { key: 'agg-badge', className: 'rf-badge' }, 'aggregated'));
        }
        if (data.permissionCard) {
          badges.push(h('span', { key: 'perm-badge', className: 'rf-badge permission' }, 'permission'));
        }
        if (data.phantom) {
          badges.push(h('span', { key: 'phantom-badge', className: 'rf-badge phantom' }, 'phantom'));
        }
        if (!data.permissionCard && (data.inlineRolePermissionCount || 0) > 0) {
          badges.push(h('span', { key: 'perm-inline-badge', className: 'rf-badge permission' }, `rules ${data.inlineRolePermissionCount}`));
        }
        if (data.podCard) {
          badges.push(h('span', { key: 'pod-badge', className: 'rf-badge pod' }, 'pod'));
        }
        if (data.workloadCard) {
          badges.push(h('span', { key: 'workload-badge', className: 'rf-badge workload' }, 'workload'));
        }
        if (data.overflowCard) {
          badges.push(h('span', { key: 'overflow-badge', className: 'rf-badge overflow' }, 'overflow'));
        }

        const handleNodes = [];
        for (let i = 0; i < EDGE_HANDLE_POSITIONS.length; i++) {
          const top = `${EDGE_HANDLE_POSITIONS[i]}%`;
          handleNodes.push(
            h(RF.Handle, {
              key: `in-left-${i}`,
              id: `in-left-${i}`,
              type: 'target',
              position: RF.Position.Left,
              className: 'rf-handle',
              style: { top }
            })
          );
          handleNodes.push(
            h(RF.Handle, {
              key: `out-right-${i}`,
              id: `out-right-${i}`,
              type: 'source',
              position: RF.Position.Right,
              className: 'rf-handle',
              style: { top }
            })
          );
        }

        const children = handleNodes.concat([
          h('div', { key: 'title-row', className: 'rf-title-row' }, [
            h('div', { key: 'title', className: 'rf-title' }, data.title || ''),
            badges.length > 0 ? h('div', { key: 'badges', style: { display: 'flex', gap: '4px', flexWrap: 'wrap' } }, badges) : null
          ]),
          h('div', { key: 'sub', className: 'rf-subtitle' }, data.subtype || ''),
          h('div', { key: 'id', className: 'rf-id' }, data.objectId || '')
        ]);

        if (data.aggregated && data.aggregationCount > 0) {
          const items = [h('div', { key: 'head', className: 'rf-agg-head' }, `aggregation sources (${data.aggregationCount})`)];
          for (let i = 0; i < data.aggregationSources.length; i++) {
            items.push(h('div', { key: `agg-${i}`, className: 'rf-agg-item' }, data.aggregationSources[i]));
          }
          if (data.aggregationHidden > 0) {
            items.push(h('div', { key: 'more', className: 'rf-agg-more' }, `+${data.aggregationHidden} more`));
          }
          children.push(h('div', { key: 'agg-list', className: 'rf-agg-list' }, items));
        }
        if (!data.permissionCard && (data.inlineRolePermissionCount || 0) > 0) {
          const items = [h('div', { key: 'head', className: 'rf-perm-head' }, `matched permissions (${data.inlineRolePermissionCount})`)];
          const permissions = Array.isArray(data.inlineRolePermissions) ? data.inlineRolePermissions : [];
          for (let i = 0; i < permissions.length; i++) {
            const permission = permissions[i] || {};
            const resourceLabel = permission.permissionNonResource
              ? (permission.name || '*')
              : formatResource(permission.permissionAPIGroup, permission.name);
            const verb = permission.permissionVerb || '*';
            const isPhantom = !!permission.phantom;
            const permTopChildren = [
              h('span', { key: `perm-verb-${i}`, className: 'rf-perm-verb' }, verb),
              h('span', { key: `perm-resource-${i}`, className: 'rf-perm-resource' }, resourceLabel)
            ];
            if (isPhantom) {
              permTopChildren.push(h('span', { key: `perm-phantom-${i}`, className: 'rf-phantom-tag' }, 'phantom'));
            }
            items.push(
              h('div', { key: `perm-${i}`, className: 'rf-perm-item' + (isPhantom ? ' is-phantom' : '') },
                h('div', { key: `perm-top-${i}`, className: 'rf-perm-item-top' }, permTopChildren)
              )
            );
          }
          if ((data.inlineRolePermissionHidden || 0) > 0) {
            items.push(h('div', { key: 'more', className: 'rf-perm-more' }, `+${data.inlineRolePermissionHidden} more`));
          }
          children.push(h('div', { key: 'perm-list', className: 'rf-perm-list' }, items));
        }

        return h('div', {
          className: 'rf-card'
            + (data.aggregated ? ' is-aggregated' : '')
            + (data.sourceLane ? ' is-source-lane' : '')
            + (data.aggregationRelationCard ? ' is-aggregation-rel' : '')
            + (data.focusDim ? ' is-focus-dim' : '')
            + (data.focusRoot ? ' is-focus-root' : '')
            + (data.permissionCard ? ' is-permission' : '')
            + (data.phantom ? ' is-phantom-card' : '')
            + (data.podCard ? ' is-pod' : '')
            + (data.workloadCard ? ' is-workload' : '')
            + (data.overflowCard ? ' is-overflow' : '')
        }, children);
      }

      const nodeTypes = { rbacCard: RBACNodeCard };

      function FlowApp() {
        const [model, setModel] = React.useState({ nodes: [], edges: [] });
        const instanceRef = React.useRef(null);

        React.useEffect(() => {
          setFlowModel = setModel;
          return () => {
            if (setFlowModel === setModel) {
              setFlowModel = null;
            }
            fitFlowView = null;
          };
        }, [setModel]);

        React.useEffect(() => {
          if (!instanceRef.current || !autoFitEl?.checked) return;
          window.requestAnimationFrame(() => {
            fitFlowInstance(instanceRef.current, 220);
          });
        }, [model]);

        return h('div', { style: { width: '100%', height: '100%' } },
          h(RF.ReactFlow, {
            nodes: model.nodes,
            edges: model.edges,
            nodeTypes,
            onNodeClick: (_evt, node) => {
              if (typeof window.__rbacgraphOnNodeClick === 'function') {
                window.__rbacgraphOnNodeClick(node.id);
              }
            },
            onPaneClick: () => {
              if (typeof window.__rbacgraphOnPaneClick === 'function') {
                window.__rbacgraphOnPaneClick();
              }
            },
            onInit: instance => {
              instanceRef.current = instance;
              fitFlowView = duration => fitFlowInstance(instance, duration);
              if (autoFitEl?.checked) {
                window.requestAnimationFrame(() => {
                  fitFlowInstance(instance, 160);
                });
              }
            },
            fitView: false,
            defaultViewport: { x: 0, y: 0, zoom: 1 },
            nodesDraggable: false,
            nodesConnectable: false,
            elementsSelectable: true,
            panOnDrag: true,
            panOnScroll: true,
            zoomOnScroll: true,
            zoomOnPinch: true,
            selectionOnDrag: false,
            minZoom: 0.45,
            maxZoom: 2.0,
            proOptions: { hideAttribution: true }
          },
            h(RF.Background, {
              variant: RF.BackgroundVariant.Dots,
              gap: 24,
              size: 1,
              color: COLORS.line
            }),
            h(RF.Controls, { showInteractive: false }),
            h(RF.MiniMap, {
              pannable: true,
              zoomable: true,
              maskColor: 'rgb(240, 240, 240, 0.65)',
              nodeStrokeWidth: 2,
              nodeStrokeColor: node => {
                if (node.data?.layer === 'AggSources') return '#e11d48';
                if (node.data?.layer === 'Aggregation') return COLORS.edgeAggregates;
                if (node.data?.layer === 'AggregatedRoles') return '#ea580c';
                if (node.data?.layer === 'Permissions') return COLORS.edgePermissions;
                if (node.data?.layer === 'WorkloadOwners') return '#1e293b';
                if (node.data?.layer === 'Pods') return COLORS.accent;
                if (node.data?.layer === 'Workloads') return COLORS.edgeOwnedBy;
                if (node.data?.aggregated) return COLORS.edgeAggregates;
                return COLORS.accent;
              },
              nodeColor: node => {
                if (node.data?.layer === 'AggSources') return COLORS.cardSourceBg;
                if (node.data?.layer === 'Aggregation') return COLORS.aggBg;
                if (node.data?.layer === 'AggregatedRoles') return '#ffedd5';
                if (node.data?.layer === 'Permissions') return COLORS.permBg;
                if (node.data?.layer === 'WorkloadOwners') return '#eef2ff';
                if (node.data?.layer === 'Pods') return COLORS.cardPodBg;
                if (node.data?.layer === 'Workloads') return COLORS.surfaceMuted;
                if (node.data?.layer === 'Subjects') return COLORS.surfaceMuted;
                return '#fffbf4';
              }
            })
          )
        );
      }

      ReactDOM.createRoot(flowRootEl).render(
        h(RF.ReactFlowProvider, null, h(FlowApp))
      );
    }

    function renderGraph(graph) {
      lastGraph = graph || { nodes: [], edges: [] };
      if (!setFlowModel) return;
      lastBaseModel = buildFlowModel(lastGraph, currentGraphOptions());
      const focusEnabled = !!focusModeEl.checked;
      const model = applyFocusToModel(lastBaseModel, lastFocusNodeID, focusEnabled);
      lastRenderModel = model;
      setFlowModel(model);
    }

    function rerenderFocusOnly() {
      if (!setFlowModel) return;
      const focusEnabled = !!focusModeEl.checked;
      const model = applyFocusToModel(lastBaseModel, lastFocusNodeID, focusEnabled);
      lastRenderModel = model;
      setFlowModel(model);
      if (lastStatus) {
        updateStats(lastStatus);
      }
    }

    function updateStats(status) {
      const graph = status.graph || { nodes: [], edges: [] };
      const allEdges = graph.edges || [];
      const aggregateLinks = allEdges.filter(edge => edge.type === 'aggregates').length;
      const sourceRoleCount = (() => {
        const nodes = graph.nodes || [];
        const nodeByID = new Map(nodes.map(node => [node.id, node]));
        const targetIDs = new Set();
        const sourceIDs = new Set();
        for (const node of nodes) {
          if (!isRoleNodeType(node.type) || !node.aggregated || !Array.isArray(node.aggregationSources)) continue;
          targetIDs.add(node.id);
          for (const rawSource of node.aggregationSources) {
            sourceIDs.add(normalizeRoleRefId(rawSource));
          }
        }
        let count = 0;
        for (const sourceID of sourceIDs) {
          const sourceNode = nodeByID.get(sourceID);
          if (!sourceNode || !isRoleNodeType(sourceNode.type) || targetIDs.has(sourceID)) continue;
          count += 1;
        }
        return count;
      })();
      let permissionVisibleCount = 0;
      let inlineRolePermissionCount = 0;
      let aggregationRelationVisibleCount = 0;
      let podsVisibleCount = 0;
      let workloadsVisibleCount = 0;
      for (const node of (lastRenderModel.nodes || [])) {
        const id = String(node.id || '');
        if (id.startsWith('permission:')) permissionVisibleCount++;
        const list = node?.data?.inlineRolePermissions;
        if (Array.isArray(list)) inlineRolePermissionCount += list.length;
        if (id.startsWith('aggregation:')) aggregationRelationVisibleCount++;
        if (node.data?.layer === 'Pods') podsVisibleCount++;
        if (node.data?.layer === 'Workloads') workloadsVisibleCount++;
      }
      statsEl.textContent = `matchedRoles=${status.matchedRoles || 0}, matchedBindings=${status.matchedBindings || 0}, matchedSubjects=${status.matchedSubjects || 0}, matchedPods=${status.matchedPods || 0}, matchedWorkloads=${status.matchedWorkloads || 0}, permissionsVisible=${permissionVisibleCount}, inlineRolePermissionsVisible=${inlineRolePermissionCount}, aggregationRelationsVisible=${aggregationRelationVisibleCount}, podsVisible=${podsVisibleCount}, workloadsVisible=${workloadsVisibleCount}, nodesVisible=${lastRenderModel.nodes.length}, edgesVisible=${lastRenderModel.edges.length}, totalNodes=${(graph.nodes || []).length}, totalEdges=${allEdges.length}, aggregateLinks=${aggregateLinks}, aggregationSourceRoles=${sourceRoleCount}, aggregateVisible=${showAggregateEdgesEl.checked ? 'on' : 'off'}, permissionsLane=${showPermissionsEl.checked ? 'on' : 'off'}, rolePermissions=${showRolePermissionsEl.checked ? 'on' : 'off'}, spreadEdges=${spreadEdgesEl.checked ? 'on' : 'off'}, focusMode=${focusModeEl.checked ? 'on' : 'off'}, runtimeView=${runtimeViewEl?.value || 'access'}, focusedNode=${lastFocusNodeID || '-'}, onlyReachable=${onlyReachableEl.checked ? 'on' : 'off'}`;
    }

    function renderChips(container, values, type) {
      container.innerHTML = (values || []).map(v => `<span class="chip ${type}">${escapeHTML(v)}</span>`).join('');
    }

    function renderResourceMap(rows) {
      resourceMapBody.innerHTML = (rows || []).map(row => `
        <tr>
          <td>${escapeHTML(row.apiGroup || '-')}</td>
          <td>${escapeHTML(row.resource || '-')}</td>
          <td>${escapeHTML(row.verb || '-')}</td>
          <td>${escapeHTML(row.roleCount)}</td>
          <td>${escapeHTML(row.bindingCount)}</td>
          <td>${escapeHTML(row.subjectCount)}</td>
        </tr>
      `).join('');
    }

    async function postQueryRequest(payloadObject) {
      const requestRaw = JSON.stringify(payloadObject);
      const headers = { 'Content-Type': 'application/json' };
      const impUser = impersonateUserEl.value.trim();
      if (impUser) headers['X-Impersonate-User'] = impUser;
      const impGroup = impersonateGroupEl.value.trim();
      if (impGroup) headers['X-Impersonate-Group'] = impGroup;
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 30000);
      try {
        const response = await fetch('/api/query', {
          method: 'POST',
          headers,
          body: requestRaw,
          signal: controller.signal
        });
        const responseRaw = await response.text();
        return {
          requestRaw,
          response,
          responseRaw,
          data: parseJSON(responseRaw)
        };
      } finally {
        clearTimeout(timeoutId);
      }
    }

    async function discoverSelectorOptions() {
      discoverOptionsBtn.disabled = true;
      setSelectorDiscoverStatus('Discovering selector values from cluster...');

      try {
        const discoverPayload = {
          spec: {
            selector: {},
            matchMode: 'any',
            wildcardMode: 'expand',
            includeRuleMetadata: false,
            includePods: false,
            includeWorkloads: false,
            podPhaseMode: 'active',
            maxPodsPerSubject: 20,
            maxWorkloadsPerPod: 10
          }
        };
        const result = await postQueryRequest(discoverPayload);
        if (!result.response.ok) {
          setSelectorDiscoverStatus(result.data?.message || result.responseRaw || `Discovery failed: HTTP ${result.response.status}`, true);
          return;
        }
        if (!result.data) {
          setSelectorDiscoverStatus('Discovery failed: API returned non-JSON response', true);
          return;
        }

        const rows = result.data?.status?.resourceMap || [];
        const options = collectSelectorOptionsFromResourceMap(rows);
        applySelectorOptions(options, true);
        setSelectorDiscoverStatus(`Loaded: verbs=${options.verbs.length}, apiGroups=${options.apiGroups.length}, resources=${options.resources.length}, nonResourceURLs=${options.nonResourceURLs.length}`);
      } catch (err) {
        setSelectorDiscoverStatus(`Discovery failed: ${String(err)}`, true);
      } finally {
        discoverOptionsBtn.disabled = false;
      }
    }

    async function run() {
      statusEl.classList.remove('error');
      statusEl.textContent = 'Running query...';
      const requestPayload = payload();
      rawStore.request = JSON.stringify(requestPayload);
      rawResponseMetaEl.textContent = 'Waiting for response...';
      rawStore.response = '';
      updateRequestText();
      updateResponseText();

      try {
        const result = await postQueryRequest(requestPayload);
        rawStore.request = result.requestRaw;
        rawStore.response = result.responseRaw;
        rawResponseMetaEl.textContent = `HTTP ${result.response.status} ${result.response.statusText}`;
        updateRequestText();
        updateResponseText();

        if (!result.response.ok) {
          statusEl.classList.add('error');
          statusEl.textContent = result.data?.message || result.responseRaw || `HTTP ${result.response.status}`;
          return;
        }

        if (!result.data) {
          statusEl.classList.add('error');
          statusEl.textContent = 'API returned non-JSON response';
          return;
        }

        const st = result.data.status || {};
        lastStatus = st;
        renderGraph(st.graph || { nodes: [], edges: [] });
        updateStats(st);
        renderChips(warningsEl, st.warnings || [], 'warn');
        renderChips(gapsEl, st.knownGaps || [], 'gap');
        renderResourceMap(st.resourceMap || []);
        applySelectorOptions(collectSelectorOptionsFromResourceMap(st.resourceMap || []), false);
        statusEl.textContent = 'Query succeeded';
      } catch (err) {
        statusEl.classList.add('error');
        statusEl.textContent = String(err);
        rawResponseMetaEl.textContent = 'Request failed';
        rawStore.response = String(err);
        updateResponseText();
      }
    }

    let queryInFlight = false;
    runBtn.addEventListener('click', async () => {
      if (queryInFlight) return;
      queryInFlight = true;
      runBtn.disabled = true;
      try { await run(); } finally {
        queryInFlight = false;
        runBtn.disabled = false;
      }
    });
    copyRequestBtn.addEventListener('click', () => copyFromElement(rawRequestEl, copyRequestBtn));
    copyResponseBtn.addEventListener('click', () => copyFromElement(rawResponseEl, copyResponseBtn));
    requestViewEl.addEventListener('change', updateRequestText);
    responseViewEl.addEventListener('change', updateResponseText);
    discoverOptionsBtn.addEventListener('click', discoverSelectorOptions);
    clearSelectorChecksBtn.addEventListener('click', () => {
      for (const kind of SELECTOR_KINDS) {
        selectorOptionSelection[kind] = new Set();
        renderSelectorOptions(kind);
      }
      updateSelectorSummary();
      setSelectorDiscoverStatus('Cleared checked selector values');
      rawStore.request = JSON.stringify(payload());
      updateRequestText();
    });
    for (const kind of SELECTOR_KINDS) {
      const listEl = SELECTOR_LIST_ELS[kind];
      const filterEl = SELECTOR_FILTER_ELS[kind];
      listEl.addEventListener('change', event => {
        const target = event.target;
        if (!(target instanceof HTMLInputElement) || target.type !== 'checkbox') return;
        const value = target.value;
        if (target.checked) {
          selectorOptionSelection[kind].add(value);
        } else {
          selectorOptionSelection[kind].delete(value);
        }
        renderSelectorOptions(kind);
        updateSelectorSummary();
        rawStore.request = JSON.stringify(payload());
        updateRequestText();
      });
      filterEl.addEventListener('input', () => {
        selectorOptionFilters[kind] = filterEl.value || '';
        renderSelectorOptions(kind);
      });
    }
    ['apiGroups', 'resources', 'verbs', 'resourceNames', 'nonResourceURLs', 'namespaceScopeNamespaces', 'maxPodsPerSubject', 'maxWorkloadsPerPod'].forEach(id => {
      const el = document.getElementById(id);
      el.addEventListener('input', () => {
        rawStore.request = JSON.stringify(payload());
        updateRequestText();
      });
    });
    ['matchMode', 'wildcardMode', 'includePods', 'includeWorkloads', 'filterPhantomAPIs', 'podPhaseMode', 'namespaceScopeStrict'].forEach(id => {
      const el = document.getElementById(id);
      el.addEventListener('change', () => {
        rawStore.request = JSON.stringify(payload());
        updateRequestText();
      });
    });

    function refreshGraphFromToggles() {
      renderGraph(lastGraph);
      if (lastStatus) {
        updateStats(lastStatus);
      }
    }

    window.__rbacgraphOnNodeClick = nodeID => {
      // Open modal for role/clusterRole nodes to show wildcard expansion.
      if (lastGraph && lastGraph.nodes) {
        const node = lastGraph.nodes.find(n => n.id === nodeID);
        if (node && isRoleNodeType(node.type)) {
          const refs = Array.isArray(node.matchedRuleRefs) ? node.matchedRuleRefs : [];
          if (refs.length > 0) {
            openRoleModal(nodeID);
          }
        }
      }
      if (!focusModeEl.checked) return;
      lastFocusNodeID = (lastFocusNodeID === nodeID) ? '' : nodeID;
      rerenderFocusOnly();
    };

    window.__rbacgraphOnPaneClick = () => {
      if (!focusModeEl.checked) return;
      if (!lastFocusNodeID) return;
      lastFocusNodeID = '';
      rerenderFocusOnly();
    };

    showAggregateEdgesEl.addEventListener('change', refreshGraphFromToggles);
    onlyReachableEl.addEventListener('change', refreshGraphFromToggles);
    showPermissionsEl.addEventListener('change', refreshGraphFromToggles);
    showRolePermissionsEl.addEventListener('change', refreshGraphFromToggles);
    spreadEdgesEl.addEventListener('change', refreshGraphFromToggles);
    runtimeViewEl.addEventListener('change', () => {
      updateRuntimeLegend();
      refreshGraphFromToggles();
    });
    laneSpacingEl.addEventListener('change', refreshGraphFromToggles);
    rowSpacingEl.addEventListener('change', refreshGraphFromToggles);
    autoFitEl.addEventListener('change', () => {
      if (autoFitEl.checked && fitFlowView) {
        fitFlowView(220);
      }
    });
    fitGraphBtn.addEventListener('click', () => {
      if (fitFlowView) {
        fitFlowView(220);
      }
    });
    focusModeEl.addEventListener('change', () => {
      if (!focusModeEl.checked) {
        lastFocusNodeID = '';
      }
      rerenderFocusOnly();
    });
    canvasHeightEl.addEventListener('change', () => {
      applyCanvasHeight(canvasHeightEl.value, true);
      refreshGraphFromToggles();
    });

    const savedCanvasHeight = (() => {
      try {
        return window.localStorage.getItem('rbacgraph.canvasHeight');
      } catch {
        return null;
      }
    })();
    for (const kind of SELECTOR_KINDS) {
      renderSelectorOptions(kind);
    }
    updateSelectorSummary();
    updateRuntimeLegend();
    applyCanvasHeight(savedCanvasHeight || canvasHeightEl.value, false);
    mountFlowApp();
    rawStore.request = JSON.stringify(payload());
    updateRequestText();
