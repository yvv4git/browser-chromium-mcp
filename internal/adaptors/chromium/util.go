package chromium

import (
	"fmt"
	"strings"

	"github.com/yvv4git/browser-chromium-mcp/internal/domain"
)

const (
	_snapshotJS = `(() => {
		const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT, null);
		let ref = 0;
		const results = [];
		function getRole(el) {
			const role = el.getAttribute('role');
			if (role) return role;
			const tag = el.tagName.toLowerCase();
			if (tag === 'a') return 'link';
			if (tag === 'button') return 'button';
			if (tag === 'input') return el.type || 'textbox';
			if (tag === 'select') return 'combobox';
			if (tag === 'textarea') return 'textbox';
			return tag;
		}
		function getName(el) {
			return el.innerText?.trim().slice(0, 80) || el.getAttribute('aria-label') || el.getAttribute('placeholder') || el.value || '';
		}
		const elements = document.querySelectorAll('a,button,input,select,textarea,[role],[onclick]');
		for (const el of elements) {
			if (el.offsetParent === null && el.tagName !== 'BODY') continue;
			ref++;
			const r = 'e' + ref;
			el.setAttribute('data-mcp-ref', r);
			results.push({ref: r, role: getRole(el), name: getName(el)});
		}
		return results.map(x => '[' + x.ref + '] ' + x.role + ' "' + x.name.replace(/"/g, "'") + '"').join('\n');
	})()`

	_linksJS = `(() => {
		const links = [];
		document.querySelectorAll('a[href]').forEach(a => {
			const href = a.href;
			if (!href) return;
			const text = (a.innerText || '').trim().slice(0, 200);
			links.push({url: href, text});
		});
		return links;
	})()`
)

func countRefs(s string) int {
	if s == "" {
		return 0
	}
	count := 0
	for _, part := range strings.Split(s, "\n") {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}
	return count
}

func mapKey(key string) (string, error) {
	switch strings.ToLower(key) {
	case "enter":
		return "\r", nil
	case "tab":
		return "\t", nil
	case "escape", "esc":
		return "\u001b", nil
	case "backspace":
		return "\b", nil
	case "delete":
		return "\u007f", nil
	case "arrowup", "up":
		return "\u001b[A", nil
	case "arrowdown", "down":
		return "\u001b[B", nil
	case "arrowleft", "left":
		return "\u001b[C", nil
	case "arrowright", "right":
		return "\u001b[D", nil
	default:
		return key, nil
	}
}

func scrollScript(direction string, amount int) (string, error) {
	if amount <= 0 {
		amount = 500
	}
	switch strings.ToLower(direction) {
	case "down":
		return fmt.Sprintf("window.scrollBy(0, %d)", amount), nil
	case "up":
		return fmt.Sprintf("window.scrollBy(0, -%d)", amount), nil
	case "right":
		return fmt.Sprintf("window.scrollBy(%d, 0)", amount), nil
	case "left":
		return fmt.Sprintf("window.scrollBy(-%d, 0)", amount), nil
	default:
		return "", domain.ErrUnsupportedDirection
	}
}
