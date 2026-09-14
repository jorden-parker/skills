# Geist component recipes

Copy-paste CSS built only from `geist-tokens.css` and `geist-type.css`. Adapt freely - these encode the step conventions so you do not have to re-derive them.

## Button

```css
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: var(--ds-size-medium);
  padding: 0 12px;
  border-radius: var(--ds-radius-base);
  font-size: 14px;
  line-height: 20px;
  font-weight: 500;
  cursor: pointer;
  transition:
    background 150ms var(--ds-motion-timing-swift),
    border-color 150ms var(--ds-motion-timing-swift);
}
.btn:focus-visible {
  outline: none;
  box-shadow: var(--ds-focus-ring);
}
.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Primary: inverted, the highest-emphasis action on the page. Use once. */
.btn-primary {
  background: var(--ds-gray-1000);
  color: var(--ds-background-100);
  border: 1px solid var(--ds-gray-1000);
}
.btn-primary:hover:not(:disabled) {
  background: var(--ds-gray-900);
  border-color: var(--ds-gray-900);
}

/* Secondary: the default button. */
.btn-secondary {
  background: var(--ds-background-100);
  color: var(--ds-gray-1000);
  border: 1px solid var(--ds-gray-400);
}
.btn-secondary:hover:not(:disabled) {
  background: var(--ds-gray-100);
  border-color: var(--ds-gray-500);
}

/* Destructive */
.btn-danger {
  background: var(--ds-red-700);
  color: #fff;
  border: 1px solid var(--ds-red-700);
}
.btn-danger:hover:not(:disabled) {
  background: var(--ds-red-800);
  border-color: var(--ds-red-800);
}
```

## Input

```css
.input {
  width: 100%;
  height: var(--ds-size-medium);
  padding: 0 10px;
  background: var(--ds-background-100);
  color: var(--ds-gray-1000);
  border: 1px solid var(--ds-gray-400);
  border-radius: var(--ds-radius-base);
  font-size: 14px;
  line-height: 20px;
}
.input::placeholder {
  color: var(--ds-gray-700);
}
.input:hover {
  border-color: var(--ds-gray-500);
}
.input:focus {
  outline: none;
  border-color: var(--ds-gray-1000);
  box-shadow: var(--ds-focus-ring);
}
.input[aria-invalid="true"] {
  border-color: var(--ds-red-700);
}
```

## Card

```css
.card {
  background: var(--ds-background-100);
  border: 1px solid var(--ds-gray-400);
  border-radius: var(--ds-radius-medium);
  padding: 20px;
}
.card-raised {
  border: none;
  box-shadow: var(--ds-shadow-border), var(--ds-shadow-small);
}
.card-header {
  font-size: 16px;
  line-height: 24px;
  letter-spacing: -0.32px;
  font-weight: 600;
  color: var(--ds-gray-1000);
}
.card-body {
  font-size: 14px;
  line-height: 20px;
  color: var(--ds-gray-900);
}
```

## Badge / status chip

The `100` / `400` / `900` pattern. Swap the hue, keep the steps.

```css
.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 12px;
  line-height: 16px;
  font-weight: 500;
  border: 1px solid;
}
.badge-neutral {
  background: var(--ds-gray-100);
  border-color: var(--ds-gray-400);
  color: var(--ds-gray-900);
}
.badge-info {
  background: var(--ds-blue-100);
  border-color: var(--ds-blue-400);
  color: var(--ds-blue-900);
}
.badge-success {
  background: var(--ds-green-100);
  border-color: var(--ds-green-400);
  color: var(--ds-green-900);
}
.badge-warning {
  background: var(--ds-amber-100);
  border-color: var(--ds-amber-400);
  color: var(--ds-amber-900);
}
.badge-error {
  background: var(--ds-red-100);
  border-color: var(--ds-red-400);
  color: var(--ds-red-900);
}
```

## Table

```css
.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
  line-height: 20px;
}
.table th {
  text-align: left;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 16px;
  font-weight: 500;
  color: var(--ds-gray-900);
  background: var(--ds-background-200);
  border-bottom: 1px solid var(--ds-gray-400);
}
.table td {
  padding: 12px;
  color: var(--ds-gray-1000);
  border-bottom: 1px solid var(--ds-gray-300);
}
.table tbody tr:hover {
  background: var(--ds-gray-100);
}
.table td.numeric {
  font-family: var(--ds-font-mono);
  font-variant-numeric: tabular-nums;
  text-align: right;
}
```

## Code block

```css
.code {
  background: var(--ds-background-200);
  border: 1px solid var(--ds-gray-400);
  border-radius: var(--ds-radius-medium);
  padding: 16px;
  overflow-x: auto;
  font-family: var(--ds-font-mono);
  font-size: 13px;
  line-height: 20px;
  color: var(--ds-gray-1000);
}
code:not(.code code) {
  background: var(--ds-gray-100);
  border-radius: 4px;
  padding: 2px 5px;
  font-family: var(--ds-font-mono);
  font-size: 0.9em;
}
```

## Page shell

```css
body {
  margin: 0;
  background: var(--ds-background-100);
  color: var(--ds-gray-1000);
  font-family: var(--ds-font-sans);
  font-size: 14px;
  line-height: 20px;
  -webkit-font-smoothing: antialiased;
}
.page {
  max-width: var(--ds-page-width);
  margin: 0 auto;
  padding: 0 24px;
}
hr {
  border: none;
  border-top: 1px solid var(--ds-gray-400);
  margin: 32px 0;
}
a {
  color: var(--ds-blue-900);
  text-decoration: none;
}
a:hover {
  text-decoration: underline;
}
```
