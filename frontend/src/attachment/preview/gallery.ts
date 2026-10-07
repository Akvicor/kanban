import PhotoSwipe from 'photoswipe'
import 'photoswipe/style.css'
import {t} from '../../i18n'

/** 画廊中的一张图片：原图地址、按方向矫正后的尺寸，以及下载文件名。 */
export interface GalleryImage {
  src: string
  width: number
  height: number
  name: string
}

/**
 * 以画廊方式浏览图片：可以缩放、左右切换，带关闭和下载按钮。打开的是原图。
 * PhotoSwipe 处理 Esc 时调用 preventDefault，卡片详情据此不会被同一次 Esc 关闭。
 */
export function openGallery(images: GalleryImage[], index: number) {
  const gallery = new PhotoSwipe({
    dataSource: images.map((image) => ({src: image.src, width: image.width, height: image.height, alt: image.name})),
    index,
    bgOpacity: 0.92,
    showHideAnimationType: 'fade',
    closeTitle: t('common.close'),
    zoomTitle: t('preview.zoom'),
    arrowPrevTitle: t('preview.prev'),
    arrowNextTitle: t('preview.next'),
    errorMsg: t('preview.imageError'),
  })
  gallery.on('uiRegister', () => {
    gallery.ui?.registerElement({
      name: 'download',
      title: t('common.download'),
      order: 8,
      isButton: true,
      tagName: 'a',
      html: `<span class="pswp-download">${t('common.download')}</span>`,
      onInit: (element, pswp) => {
        const link = element as HTMLAnchorElement
        const update = () => {
          const image = images[pswp.currIndex]
          link.href = image.src
          link.download = image.name
        }
        update()
        pswp.on('change', update)
      },
    })
    gallery.ui?.registerElement({
      name: 'caption',
      order: 9,
      isButton: false,
      appendTo: 'root',
      onInit: (element, pswp) => {
        const update = () => {
          element.textContent = images[pswp.currIndex]?.name ?? ''
        }
        update()
        pswp.on('change', update)
      },
    })
  })
  gallery.init()
}
