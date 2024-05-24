---
categories: ["article"]
tags: ["hugo", "chartjs", "static-websites", "blog"]
date: "2024-05-28"
description: "Let's find out how to add chart.js to your static website built with Hugo."
cover: "cover.jpg"
images: ["/posts/hugo-chartjs/cover.jpg"]
featuredpath: "date"
title: "Adding chart.js to Hugo"
slug: "hugo-chartjs"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/hugo-chartjs
---

I recently wanted to add some charts to a blog post and [mermaid](https://mermaid.live) just wasn't cutting it. Mermaid didn't support the options I wanted to use and ultimately wasn't flexible enough to show a horizontal bar chart with the customization options I wanted. So I went looking for alternatives... and that's when I found this [shen-yu/hugo-chart](https://github.com/shen-yu/hugo-chart)... layout? for hugo that adds [Chart.js](https://www.chartjs.org/) support. Chart.js is a great javascript library for creating many kinds of charts with many customization options. *Perfect*, I thought. As I started using `shen/yu-hugo-chart` to add chart.js to my site, a few things stood out to me:

- I felt like adding a "layout" and a new git submodule just for this was fairly extreme just to add support for a single javascript library.
- The documentation on the [chart.js website](https://www.chartjs.org/) didn't match up with what was supported. I then realized that the hugo-chart plugin was using *Chart.js version 2* when the latest released version was v4... a whole two major versions behind. Yikes. So not only would this add a new git submodule, which is annoying, but it wasn't even up-to-date.

At this point, I was ready to throw the entire project aside and do my own thing. This is what I ended up with. It only requires a single file to add the new shortcode. So instead of having a git dependency, just add this file to your shortcodes.

### Installation
Add this file at the path `layouts/shortcodes/chart.html`:
```html
{{- if not (.Page.Scratch.Get "hasChartJS") -}}
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script> Chart.defaults.color = '#fff'; </script>
{{- .Page.Scratch.Set "hasChartJS" true -}}
{{- end -}}

{{- $id := substr (md5 .Inner) 0 16 -}}
<div class="chart">
    <canvas id="{{ $id }}"></canvas>
</div>
<script>
    document.addEventListener('DOMContentLoaded', () => {
        var ctx = document.getElementById('{{ $id }}')
        var options = {{ .Inner | chomp | safeJS }};
        new Chart(ctx, options);
    });
</script>
```

The `.Page.Scratch.Set` and `.Page.Scratch.Get` calls allows me to only have the `<script>` tag that imports chart.js a single time, which seems cleaner to me than importing it for each call to the chart shortcode.

Once this file is in place, you can now call the `chart` shortcode like this:
```javascript
{{</* chart */>}}
{
  type: 'type',
  data: {...},
  options: {...}
}
{{</* /chart */>}}
```
and a chart will be generated using chart.js. A few full examples are below:

### Line Chart
```javascript
{{</* chart */>}}
{
  type: 'line',
  data: {
      labels: [
        'Jan',
        'Feb',
        'Mar',
        'Apr',
        'May',
        'June',
        'Juny'
      ],
    datasets: [{
      label: 'My First Dataset',
      data: [65, 59, 80, 81, 56, 55, 40],
      fill: false,
      borderColor: 'rgb(75, 192, 192)',
      tension: 0.1
    }]
  },
}
{{</* /chart */>}}
```

{{< chart >}}
{
  type: 'line',
  data: {
      labels: [
        'Jan',
        'Feb',
        'Mar',
        'Apr',
        'May',
        'June',
        'Juny'
      ],
    datasets: [{
      label: 'My First Dataset',
      data: [65, 59, 80, 81, 56, 55, 40],
      fill: false,
      borderColor: 'rgb(75, 192, 192)',
      tension: 0.1
    }]
  },
}
{{< /chart >}}
See more options on line charts [here](https://www.chartjs.org/docs/latest/charts/line.html).

### Bar Chart
```javascript
{{</* chart */>}}
{
    type: 'bar',
    data: {
        labels: [
          'Dataset 1',
          'Dataset 2',
          'Dataset 3',
          'Dataset 4',
          'Dataset 5',
          'Dataset 6'
        ],
        datasets: [{
            label: 'units',
            data: [36, 50, 3, 36, 63, 79]
        }]
    },
    options: {
        indexAxis: 'y',
        plugins: {
            legend: {
                display: false
            },
            title: {
                display: true,
                text: 'Bar Chart title'
            }
        }
    }
}
{{</* /chart */>}}
```

{{< chart >}}
{
    type: 'bar',
    data: {
        labels: [
          'Dataset 1',
          'Dataset 2',
          'Dataset 3',
          'Dataset 4',
          'Dataset 5',
          'Dataset 6'
        ],
        datasets: [{
            label: 'units',
            data: [36, 50, 3, 36, 63, 79]
        }]
    },
    options: {
        indexAxis: 'y',
        plugins: {
            legend: {
                display: false
            },
            title: {
                display: true,
                text: 'Bar Chart title'
            }
        }
    }
}
{{< /chart >}}
See more options on bar charts [here](https://www.chartjs.org/docs/latest/charts/bar.html).

### Polar Area Chart
You can use any chart [available in chart.js](https://www.chartjs.org/docs/latest/). Here's a polar area chart:
```javascript
{{</* chart */>}}
{
  type: 'polarArea',
  data: {
    labels: [
      'Red',
      'Green',
      'Yellow',
      'Grey',
      'Blue'
    ],
    datasets: [{
      label: 'My First Dataset',
      data: [11, 16, 7, 3, 14],
      backgroundColor: [
        'rgb(255, 99, 132)',
        'rgb(75, 192, 192)',
        'rgb(255, 205, 86)',
        'rgb(201, 203, 207)',
        'rgb(54, 162, 235)'
      ]
    }]
  },
  options: {}
}
{{</* /chart */>}}
```

{{< chart >}}
{
  type: 'polarArea',
  data: {
    labels: [
      'Red',
      'Green',
      'Yellow',
      'Grey',
      'Blue'
    ],
    datasets: [{
      label: 'My First Dataset',
      data: [11, 16, 7, 3, 14],
      backgroundColor: [
        'rgb(255, 99, 132)',
        'rgb(75, 192, 192)',
        'rgb(255, 205, 86)',
        'rgb(201, 203, 207)',
        'rgb(54, 162, 235)'
      ]
    }]
  },
  options: {}
}
{{< /chart >}}
See more options on bar charts [here](https://www.chartjs.org/docs/latest/charts/polar.html).

### Some other kinds
I'm going to spare you from giving an example of every type of chart. Please reference the official [chart.js documentation](https://www.chartjs.org/docs/latest/) to see all available options.


### End
I introduced a way to add a fully up-to-date chart.js to your hugo website that avoids many of the downsides of existing solutions. If you liked this post, feel free to send me any comments or questions [on mastodon](https://infosec.exchange/@sudorandom)!
