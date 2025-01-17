# FE setup and configuration

## Tailwind and DaisyUI development experience in Visual Studio Code

 1. Install the `Tailwind CSS IntelliSense` extension in Visual Studio Code to get IntelliSense autocompletion for Tailwind CSS classes
 2. Run in terminal

    ```bash
    npm install -D tailwindcss
    npm install @tailwindcss/forms
    npm install @tailwindcss/typography
    npm i -D daisyui@latest
    ```

 3. Add the following code to the `settings.json` file in Visual Studio Code

    ```json
    "editor.quickSuggestions": {
      "strings": true
    }
    ```

## Why?
### `Webpack` and `Parcel`

> Webpack and Parcel combine all your JavaScript and CSS files into single or multiple bundles, reducing the number of HTTP requests needed to load your application, which can significantly improve load times.

### Why Use `PostCSS` with `TailwindCSS`?
> Performance Optimization: PostCSS processes your TailwindCSS files to **remove unused styles with PurgeCSS** (now integrated into TailwindCSS v2.0 and above), **significantly reducing the size** of your final CSS file for production.
>
> Customization and Extensibility: PostCSS allows you to use other plugins alongside TailwindCSS and Autoprefixer, enabling you to further optimize and enhance your CSS (e.g., **minifying CSS, custom properties processing**).

## 1. Install Node.js

- Download and install Node.js from [Node.js](https://nodejs.org/en/)
- Verify the installation by running the following commands in the terminal:

  ```bash
  node -v
  npm -v
  ```

## 2. Init the project with HTMX, DaisyUI, TailwindCSS, PostCSS and Autoprefixer

- Init and install the project with the following commands:

  ```bash
  npm init -y
  npm install tailwindcss postcss autoprefixer
  npx tailwindcss init
  npm install daisyui
  npm install --save-dev webpack webpack-cli css-loader style-loader parcel-bundler
  ```

- Add the following code to the `tailwind.config.js` file:

  ```javascript
  /** @type {import('tailwindcss').Config} */
  module.exports = {
    content: [],
    theme: {
      container: {
        center: true,
        padding: "1rem",
      },
    },
    daisyui: {
      themes: ["dark", "cupcake"],
    },
    plugins: [
      require("@tailwindcss/forms"),
      require("@tailwindcss/typography"),
      require("daisyui"),
    ],
  };
    ```

- Add the following code to the `postcss.config.js` file:

  ```javascript
  module.exports = {
  plugins: [
    require('tailwindcss'),
    require('autoprefixer'),
  ],
  };
  ```

- Add the following code to the `styles.css` file:

  ```css
  @tailwind base;
  @tailwind components;
  @tailwind utilities;
  @import 'daisyui';
  ```

- `packege.json` example

  ```json
  {
  "name": "your-project",
  "version": "1.0.0",
  "scripts": {
    "build-css": "postcss src/styles.css -o dist/styles.css",
    "build-css-w": "postcss src/styles.css -o dist/styles.css --watch",
    "build": "webpack --mode production",
    "start": "webpack serve --mode development --open",
    "test": "echo \"Error: no test specified\" && exit 1"
  },
  "devDependencies": {
    "webpack": "^x.x.x",
    "webpack-cli": "^x.x.x",
    "css-loader": "^x.x.x",
    "style-loader": "^x.x.x"
  }
  }

  ```

### Parcel using
Simply point Parcel to your entry file (e.g., an HTML file linking to your CSS and JavaScript), and it takes care of the rest:

```bash
npx parcel index.html
```

## 3. Init the project with HTMX, DaisyUI, TailwindCSS, PostCSS and Autoprefixer

- Create a new project with the following command:

  ```bash
  npx degit tailwindlabs/tailwindcss-starter#main my-htmx-project
  cd my-htmx-project
  npm install
  ```

- Add the following dependencies to the `package.json` file:

  ```json
    "dependencies": {
        "htmx.org": "^1.5.0",
        "daisyui": "^1.10.0"
    }
    ```

- Add the following scripts to the `package.json` file:

- ```json
  "scripts": {
    "start": "postcss styles.css -o public/build/tailwind.css -w",
    "build": "postcss styles.css -o public/build/tailwind.css"
  }
  ```

- Create a new file called `htmx.js` in the `public` directory and add the following code:

  ```javascript
    document.addEventListener("htmx:load", function(event) {
        // Add your custom JavaScript here
    })
    ```

- Add the following code to the `styles.css` file:

- ```css
  @import 'tailwindcss/base';
  @import 'tailwindcss/components';
  @import 'tailwindcss/utilities';
  @import 'daisyui';
  ```
