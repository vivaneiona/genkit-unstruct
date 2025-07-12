# Stick Template Engine Example

This example demonstrates how to use the [Stick template engine](https://github.com/tyler-sommer/stick) with unstract for advanced prompt templating.

## Features

Stick provides Twig-compatible templating with advanced features:

- **Conditional Logic**: `{% if %}`, `{% else %}`, `{% endif %}`
- **Loops**: `{% for item in items %}` with `loop.index`, `loop.first`, `loop.last`
- **Variables**: `{% set name = value %}`
- **Filters**: `{{ value|filter }}`
- **Comments**: `{# This is a comment #}`
- **Template Inheritance**: `{% extends %}`, `{% block %}`

## Template Structure

Templates are stored in the `templates/` directory:

- `code.twig` - Project code extraction with loops and conditionals
- `cert.twig` - Certificate issuer extraction with variable assignment
- `extractor.twig` - Coordinate extraction with math operations and validation

## Running the Example

1. Set your Google AI API key:
   ```bash
   export GEMINI_API_KEY=your_api_key_here
   ```

2. Run the example:
   ```bash
   just do stick run
   ```

   Or directly:
   ```bash
   cd examples/stick
   go run main.go
   ```

## Template Examples

### Basic Loop and Conditionals

```twig
{% set instructions = ["Read document", "Extract data", "Return JSON"] %}

Instructions:
{% for instruction in instructions %}
{{ loop.index }}. {{ instruction }}
{% endfor %}

{% if version > 1 %}
Enhanced mode enabled (version {{ version }})
{% endif %}
```

### Dynamic JSON Generation

```twig
Expected format:
{
  {% for key in "{{.Keys}}".split(',') %}
  "{{ key.trim() }}": "value"{% if not loop.last %},{% endif %}
  {% endfor %}
}
```

### Variable Assignment and Patterns

```twig
{% set company_patterns = ["Inc", "Corp", "Ltd", "LLC"] %}

Search for companies ending with:
{% for pattern in company_patterns %}
- {{ pattern }}
{% endfor %}
```

## Advantages of Stick Templates

1. **Advanced Logic**: Complex conditionals and loops
2. **Reusability**: Template inheritance and includes
3. **Maintainability**: Clear separation of logic and content
4. **Validation**: Built-in template syntax validation
5. **Performance**: Compiled templates for faster execution

## Comparison with Simple Templates

| Feature | Simple Map | Stick Templates |
|---------|------------|-----------------|
| Conditionals | ❌ | ✅ |
| Loops | ❌ | ✅ |
| Variables | ❌ | ✅ |
| Inheritance | ❌ | ✅ |
| Comments | ❌ | ✅ |
| Validation | ❌ | ✅ |

## Advanced Usage

See the template files for examples of:
- Loop variables (`loop.index`, `loop.first`, `loop.last`)
- String operations (`split()`, `trim()`)
- Mathematical expressions
- Multi-line template organization
- Complex JSON structure generation
