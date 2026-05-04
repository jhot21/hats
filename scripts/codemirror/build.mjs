import { build } from 'esbuild'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))

await build({
  stdin: {
    contents: `
      import { EditorView, basicSetup } from 'codemirror'
      import { yaml } from '@codemirror/lang-yaml'
      import { json } from '@codemirror/lang-json'
      window.CM = { EditorView, basicSetup, yaml, json }
    `,
    resolveDir: __dirname,
  },
  bundle: true,
  minify: true,
  outfile: resolve(__dirname, '../../internal/admin/static/codemirror.min.js'),
  format: 'iife',
})

console.log('CodeMirror bundle written.')
