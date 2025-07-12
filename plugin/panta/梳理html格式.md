# 91panta HTML结构分析

这是一个名为"盘Ta-云盘资源共享站"的网页，搜索关键词为"瑞克和莫蒂"的结果页面。HTML结构如下：

## 文档结构
- `<!DOCTYPE html>` 声明为XHTML 1.0 Transitional
- `<html>` 根元素
- `<head>` 包含元数据和资源引用
- `<body>` 包含页面内容

## 头部区域
- `<head>` 包含：
  - `<base>` 设置基础URL为"https://www.91panta.cn/"
  - `<meta>` 设置字符集和SEO相关内容
  - `<title>` 页面标题："盘Ta-云盘资源共享站"
  - CSS和JavaScript文件引用

## 页面主体
1. **头部模块** (`<div class="headerModule">`)：
   - 网站logo
   - 导航栏（首页、互助区、帮助中心）
   - 搜索框（当前搜索"瑞克和莫蒂"）
   - 菜单区（注册、登录按钮）

2. **主内容区** (`<div class="skeleton">`)：
   - 主内容容器 (`<div class="main wrap">`)
   - 话题模块 (`<div class="topicModule">`)
     - 话题列表 (`<div class="topicList">`)
     - 多个话题项 (`<div class="topicItem">`)，每个包含：
       - 头像区 (`<div class="avatarBox">`)
       - 内容区 (`<div class="content">`)，包含：
         - 信息栏：话题类型、标签、用户名、发布时间
         - 标题：链接到具体内容
         - 摘要：显示内容预览和云盘链接
       - 统计区 (`<div class="statistic">`)：显示浏览量和评论数

3. **分页区域** (`<div class="topicPage">`)：
   - 显示"共15079条"
   - 页码导航（当前第1页，共754页）
   - 页码输入框和跳转功能

4. **页脚** (`<div class="footer">`)：
   - 版权信息和免责声明

这个页面主要展示搜索"瑞克和莫蒂"的结果，每个结果项包含相关资源的中国移动云盘链接。页面设计简洁，主要功能是资源分享和搜索。

## 帖子详情页面结构

帖子详情页面（posts目录下的txt文件）的HTML结构如下：

1. **文档结构**：
   - 与搜索结果页面相同，使用XHTML 1.0 Transitional

2. **头部区域**：
   - 与搜索结果页面类似，但`<title>`标签内容包含帖子标题
   - 加载了更多的JS库，如DPlayer（视频播放器）、KindEditor（富文本编辑器）、Layer（弹层组件）等

3. **页面主体**：
   - **头部模块** (`<div class="headerModule">`)：与搜索结果页面相同
   
   - **主内容区** (`<div class="skeleton">`)：
     - 主内容容器 (`<div class="main wrap">`)
     - 话题内容模块 (`<div class="topicContentModule">`)
       - 左侧内容区 (`<div class="left">`)
         - 话题包装 (`<div class="topic-wrap">`)
           - 话题标签 (`<div class="topicTag">`)：显示分类标签（如"动漫"）
           - 话题框 (`<div class="topicBox">`)
             - 标题 (`<div class="title">`)：完整的帖子标题
             - 话题信息 (`<div class="topicInfo">`)：发布时间、阅读次数、评论数
             - 话题内容 (`<div class="topicContent">`)：
               - 资源链接（中国移动云盘链接）
               - 详细描述（如导演、演员、简介等）
               - 图片（资源封面或截图）
               - 可能包含隐藏内容区域（需要密码、评论或积分才能查看）
             - 收藏/点赞区 (`<div class="favorite-formModule">`)：
               - 收藏按钮及计数
               - 点赞按钮及计数
               - 相关JavaScript功能
       - 可能包含评论区（如果有评论）
       - 可能包含相关推荐区域

4. **页脚**：
   - 与搜索结果页面相同

帖子详情页面主要展示特定资源的详细信息，包括资源描述、云盘链接、封面图片等。页面结构清晰，重点突出资源信息和下载链接。 